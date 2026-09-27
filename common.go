package apin

import "context"

// Closer is the lifecycle contract every Shell must satisfy so callers can
// release resources uniformly on shutdown.
type Closer interface {
	Close(ctx context.Context) error
}
