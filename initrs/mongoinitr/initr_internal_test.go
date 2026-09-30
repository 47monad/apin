package mongoinitr

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// unreachableURI points at a port nothing listens on and fails server
// selection quickly, so New's ping fails without a live MongoDB.
const unreachableURI = "mongodb://127.0.0.1:65534/?connect=direct&serverSelectionTimeoutMS=50"

func TestNewDisconnectsClientWhenPingFails(t *testing.T) {
	original := disconnect
	disconnected := false
	disconnect = func(client *mongo.Client, ctx context.Context) error {
		disconnected = true
		return original(client, ctx)
	}
	defer func() { disconnect = original }()

	_, err := New(context.Background(),
		WithURI(unreachableURI),
		WithTimeout(50*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected a ping failure for an unreachable server")
	}
	if !disconnected {
		t.Fatal("expected the client to be disconnected after a ping failure")
	}
	if !strings.Contains(err.Error(), "problem pinging database") {
		t.Fatalf("ping error chain was lost: %v", err)
	}
}

func TestNewDisconnectsClientWhenPingFailsWithCanceledContext(t *testing.T) {
	original := disconnect
	disconnected := false
	disconnect = func(client *mongo.Client, ctx context.Context) error {
		disconnected = true
		return original(client, ctx)
	}
	defer func() { disconnect = original }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := New(ctx, WithURI(unreachableURI))
	if err == nil {
		t.Fatal("expected a ping failure for a canceled context")
	}
	if !disconnected {
		t.Fatal("expected cleanup even when the caller context is already canceled")
	}
}

func TestNewJoinsPingAndCleanupErrors(t *testing.T) {
	original := disconnect
	cleanupErr := errors.New("cleanup failed")
	disconnect = func(client *mongo.Client, ctx context.Context) error {
		_ = original(client, ctx)
		return cleanupErr
	}
	defer func() { disconnect = original }()

	_, err := New(context.Background(),
		WithURI(unreachableURI),
		WithTimeout(50*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected joined ping and cleanup errors")
	}
	if !errors.Is(err, cleanupErr) {
		t.Fatalf("cleanup error chain was lost: %v", err)
	}
	if !strings.Contains(err.Error(), "problem pinging database") {
		t.Fatalf("ping error chain was lost: %v", err)
	}
}
