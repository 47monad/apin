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
