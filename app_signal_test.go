package apin

import (
	"context"
	"syscall"
	"testing"
	"time"
)

func TestSecondSignalCancelsCleanupWithoutExitingProcess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stop := make(chan struct{})
	defer close(stop)

	app := New()
	app.watchForceCancel(stop, cancel)

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("failed to send second signal: %v", err)
	}

	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("second signal did not cancel cleanup")
	}
}
