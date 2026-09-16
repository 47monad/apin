package apin

import (
	"context"

	"github.com/go-logr/logr"
)

type LoggerShell struct {
	Logger logr.Logger
}

// Closer is the lifecycle contract every Shell must satisfy so callers can
// release resources uniformly on shutdown.
type Closer interface {
	Close(ctx context.Context) error
}
