package apin

import "context"

// Closer is the lifecycle contract every Shell must satisfy so callers can
// release resources uniformly on shutdown.
type Closer interface {
	Close(ctx context.Context) error
}

// ReadinessChecker is the connectivity contract a Shell can satisfy so callers
// can probe infrastructure uniformly before serving traffic. Ready reports the
// current state of the underlying resource; it does not wait for recovery.
// Callers bound the probe by passing a context with a deadline.
type ReadinessChecker interface {
	Ready(ctx context.Context) error
}
