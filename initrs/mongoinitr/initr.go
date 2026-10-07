package mongoinitr

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Shell struct {
	Client *mongo.Client
	DB     *mongo.Database
}

const defaultPingTimeout = 10 * time.Second

// disconnect releases a native client. It is a package-level function so tests
// can observe and inject startup cleanup.
var disconnect = (*mongo.Client).Disconnect

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store, err := resolveConfig(opts...)
	if err != nil {
		return nil, err
	}

	client, err := mongo.Connect(store.clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, store.pingTimeout)
	defer cancel()

	if err = client.Ping(pingCtx, nil); err != nil {
		pingErr := fmt.Errorf("problem pinging database: %w", err)
		// Disconnect with a fresh, bounded context: the ping context may
		// already be expired, but the client still owns resources to release.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), store.pingTimeout)
		cleanupErr := disconnect(client, cleanupCtx)
		cleanupCancel()
		if cleanupErr != nil {
			return nil, errors.Join(pingErr, fmt.Errorf("failed to disconnect from mongodb: %w", cleanupErr))
		}
		return nil, pingErr
	}

	shell := &Shell{Client: client}
	if store.dbName != "" {
		shell.DB = client.Database(store.dbName)
	}

	return shell, nil
}

func resolveConfig(opts ...Option) (*resolvedConfig, error) {
	store := &resolvedConfig{
		clientOptions: options.Client(),
		pingTimeout:   defaultPingTimeout,
	}
	if err := apply(store, opts); err != nil {
		return nil, err
	}
	if err := store.clientOptions.Validate(); err != nil {
		return nil, fmt.Errorf("invalid MongoDB configuration: %w", err)
	}
	if store.pingTimeout <= 0 {
		return nil, fmt.Errorf("invalid MongoDB configuration: ping timeout must be positive")
	}
	return store, nil
}

// Ready verifies MongoDB connectivity with a ping. It satisfies
// apin.ReadinessChecker. Callers can bound the probe by passing a context with
// a deadline.
func (shell *Shell) Ready(ctx context.Context) error {
	if shell.Client == nil {
		return errors.New("mongoinitr: shell is not initialized")
	}
	if err := shell.Client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("problem pinging database: %w", err)
	}
	return nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := disconnect(shell.Client, ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %w", err)
	}
	return nil
}
