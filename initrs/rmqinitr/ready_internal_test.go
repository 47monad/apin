package rmqinitr

import (
	"context"
	"errors"
	"testing"
)

// Ready reports the current state immediately: it never waits for a reconnect.
func TestReadyReportsCurrentState(t *testing.T) {
	shell := &Shell{}
	if err := shell.Ready(context.Background()); !errors.Is(err, ErrNotHealthy) {
		t.Fatalf("Ready() on an unhealthy shell = %v, want ErrNotHealthy", err)
	}

	shell.healthy = true
	if err := shell.Ready(context.Background()); err != nil {
		t.Fatalf("Ready() on a healthy shell = %v, want nil", err)
	}

	shell.closed = true
	if err := shell.Ready(context.Background()); !errors.Is(err, ErrShellClosed) {
		t.Fatalf("Ready() on a closed shell = %v, want ErrShellClosed", err)
	}
}
