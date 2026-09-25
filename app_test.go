package apin_test

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/47monad/apin"
)

type fakeShell struct {
	id      string
	err     error
	records *[]string
	mu      *sync.Mutex
}

func (s *fakeShell) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	*s.records = append(*s.records, s.id)
	return s.err
}

func TestCloseReverseOrder(t *testing.T) {
	var records []string
	mu := &sync.Mutex{}

	app := apin.NewApp()
	shells := []*fakeShell{
		{id: "a", records: &records, mu: mu},
		{id: "b", records: &records, mu: mu},
		{id: "c", records: &records, mu: mu},
	}
	app.Track(shells[0], shells[1], shells[2])

	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if got, want := records, []string{"c", "b", "a"}; len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("close order = %v, want %v", got, want)
	}

	// Second Close is a no-op.
	if err := app.Close(context.Background()); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestCloseJoinsErrors(t *testing.T) {
	var records []string
	mu := &sync.Mutex{}

	wantErr := errors.New("boom")
	app := apin.NewApp()
	app.Track(
		&fakeShell{id: "a", records: &records, mu: mu},
		&fakeShell{id: "b", err: wantErr, records: &records, mu: mu},
	)

	err := app.Close(context.Background())
	if err == nil {
		t.Fatal("Close() error = nil, want joined error")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("Close() error = %v, want wrapped %v", err, wantErr)
	}
}

func TestRunClosesOnContextCancel(t *testing.T) {
	var records []string
	mu := &sync.Mutex{}

	app := apin.NewApp()
	app.Track(&fakeShell{id: "shell", records: &records, mu: mu})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := app.Run(ctx, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(records) != 1 {
		t.Errorf("shell not closed by Run: %v", records)
	}
}

func TestRunReturnsRunnableError(t *testing.T) {
	var records []string
	mu := &sync.Mutex{}

	app := apin.NewApp()
	app.Track(&fakeShell{id: "shell", records: &records, mu: mu})

	wantErr := errors.New("runnable failed")
	err := app.Run(context.Background(), func(ctx context.Context) error { return wantErr })
	if !errors.Is(err, wantErr) {
		t.Errorf("Run() error = %v, want %v", err, wantErr)
	}
	if len(records) != 1 {
		t.Errorf("shell not closed after runnable failure: %v", records)
	}
}

// slowCloser blocks in Close until its context is done, so a shutdown
// deadline is what releases it.
type slowCloser struct{}

func (slowCloser) Close(ctx context.Context) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestSetShutdownTimeout(t *testing.T) {
	app := apin.NewApp()
	if got, want := app.ShutdownTimeout(), 30*time.Second; got != want {
		t.Errorf("default ShutdownTimeout() = %v, want %v", got, want)
	}

	if got := app.SetShutdownTimeout(50 * time.Millisecond).ShutdownTimeout(); got != 50*time.Millisecond {
		t.Errorf("ShutdownTimeout() = %v, want %v", got, 50*time.Millisecond)
	}
	// Non-positive values fall back to the default rather than closing with
	// an already-expired context.
	if got := app.SetShutdownTimeout(0).ShutdownTimeout(); got != 30*time.Second {
		t.Errorf("ShutdownTimeout() after SetShutdownTimeout(0) = %v, want %v", got, 30*time.Second)
	}
}

func TestSetShutdownTimeoutBoundsShutdownPhase(t *testing.T) {
	var records []string
	mu := &sync.Mutex{}

	app := apin.NewApp()
	app.SetShutdownTimeout(50 * time.Millisecond)
	app.Track(
		&fakeShell{id: "fast", records: &records, mu: mu},
		slowCloser{},
		&fakeShell{id: "after", records: &records, mu: mu},
	)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	err := app.Run(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Run() error = %v, want the shutdown deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Errorf("Run() took %v, want it bounded by the shutdown timeout", elapsed)
	}
	// Shells close in reverse tracking order, and the slow shell is only
	// released by the deadline, so the one after it still gets closed.
	if len(records) != 2 || records[0] != "after" || records[1] != "fast" {
		t.Errorf("close order = %v, want [after fast]", records)
	}
}

func TestRunStopsOnSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal self-delivery requires unix")
	}

	var records []string
	mu := &sync.Mutex{}

	app := apin.NewApp()
	app.Track(&fakeShell{id: "shell", records: &records, mu: mu})

	runCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(50 * time.Millisecond)
		if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
			t.Errorf("failed to send SIGINT: %v", err)
		}
	}()

	err := app.Run(runCtx, func(ctx context.Context) error {
		<-ctx.Done()
		return nil
	})
	if err != nil {
		t.Fatalf("Run() error = %v, want clean shutdown", err)
	}
	if len(records) != 1 {
		t.Errorf("shell not closed after signal: %v", records)
	}
}
