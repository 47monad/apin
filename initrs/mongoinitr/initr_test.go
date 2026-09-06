package mongoinitr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/mongoinitr"
)

func TestErrorWrapping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Use an invalid URI that fails fast
	builder := mongoinitr.Opts().
		SetURI("mongodb://127.0.0.1:65534/?connect=direct&serverSelectionTimeoutMS=100").
		SetTimeout(100 * time.Millisecond)

	_, err := mongoinitr.New(ctx, builder)
	if err == nil {
		t.Fatal("expected error connecting/pinging invalid mongo URI, got nil")
	}

	// Verify the error is wrapped with %w and can be unwrapped
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatalf("expected error to be unwrappable, but errors.Unwrap(err) returned nil. err: %v", err)
	}
}
