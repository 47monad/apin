package httpinitr

import (
	"context"
	"net/http"
	"testing"
)

// Listen records its listener so Close can release it before Serve starts; a
// following Serve must reuse that record instead of tracking it twice.
func TestListenThenServeTracksListenerOnce(t *testing.T) {
	shell, err := New(context.Background(), WithHandler(http.NotFoundHandler()))
	if err != nil {
		t.Fatal(err)
	}
	shell.Server.Addr = "127.0.0.1:0"

	listener, err := shell.Listen()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	serveDone := make(chan error, 1)
	go func() { serveDone <- shell.Serve(ctx, listener) }()

	// A served request proves Serve got past listener tracking.
	response, err := http.Get("http://" + listener.Addr().String())
	if err != nil {
		t.Fatalf("GET() error = %v", err)
	}
	_ = response.Body.Close()

	shell.mu.Lock()
	tracked := len(shell.listeners)
	shell.mu.Unlock()
	if tracked != 1 {
		t.Fatalf("tracked listeners = %d, want 1", tracked)
	}

	cancel()
	if err := <-serveDone; err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
