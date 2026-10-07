package redisinitr

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// Shell holds a go-redis universal client covering standalone, Sentinel, and
// cluster deployments.
type Shell struct {
	Client redis.UniversalClient
}

// The shell satisfies the readiness contract without importing apin.
var _ interface{ Ready(context.Context) error } = (*Shell)(nil)

// MustNew is New but panics on failure.
func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

// New validates the configuration and constructs a universal client lazily:
// it does not connect. Use Ready to verify connectivity.
func New(_ context.Context, opts ...Option) (*Shell, error) {
	config := &resolvedConfig{opts: &redis.UniversalOptions{}}
	if err := apply(config, opts); err != nil {
		return nil, err
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	return &Shell{Client: redis.NewUniversalClient(config.opts)}, nil
}

// Ready pings the server. It satisfies apin.ReadinessChecker. Callers can bound
// the probe by passing a context with a deadline.
func (shell *Shell) Ready(ctx context.Context) error {
	if shell.Client == nil {
		return fmt.Errorf("redisinitr: shell is not initialized")
	}
	if err := shell.Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redisinitr: ping failed: %w", err)
	}
	return nil
}

// Close releases the client.
func (shell *Shell) Close(context.Context) error {
	if shell.Client == nil {
		return nil
	}
	if err := shell.Client.Close(); err != nil {
		return fmt.Errorf("redisinitr: close failed: %w", err)
	}
	return nil
}
