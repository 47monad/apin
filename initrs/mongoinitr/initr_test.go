package mongoinitr_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/mongoinitr"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestReadyNotInitialized(t *testing.T) {
	shell := &mongoinitr.Shell{}
	if err := shell.Ready(context.Background()); err == nil {
		t.Fatal("Ready() error = nil on uninitialized shell, want error")
	}
}

func TestReadyReportsUnreachableServer(t *testing.T) {
	client, err := mongo.Connect(options.Client().ApplyURI(
		"mongodb://127.0.0.1:65534/?connect=direct&serverSelectionTimeoutMS=50",
	))
	if err != nil {
		t.Fatalf("mongo.Connect() error = %v", err)
	}
	defer func() { _ = client.Disconnect(context.Background()) }()

	shell := &mongoinitr.Shell{Client: client}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := shell.Ready(ctx); err == nil {
		t.Fatal("Ready() error = nil for an unreachable server, want error")
	}
}

func TestErrorWrapping(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Use an invalid URI that fails fast
	_, err := mongoinitr.New(ctx,
		mongoinitr.WithURI("mongodb://127.0.0.1:65534/?connect=direct&serverSelectionTimeoutMS=100"),
		mongoinitr.WithTimeout(100*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected error connecting/pinging invalid mongo URI, got nil")
	}

	// Verify the error is wrapped with %w and can be unwrapped
	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		t.Fatalf("expected error to be unwrappable, but errors.Unwrap(err) returned nil. err: %v", err)
	}
}
