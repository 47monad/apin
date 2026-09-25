package closr

import (
	"context"

	"github.com/47monad/apin"
)

// Closer is the lifecycle contract for shells. Aliased from the apin core so
// existing references keep working; prefer apin.Closer in new code.
type Closer = apin.Closer

type Options struct {
	ctx context.Context
}

type Option func(o *Options)

func WithContext(ctx context.Context) Option {
	return func(o *Options) {
		o.ctx = ctx
	}
}

func Close(closer Closer, opts ...Option) error {
	_opts := &Options{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		opt(_opts)
	}
	return closer.Close(_opts.ctx)
}

// MustClose closes closer and panics if it fails, following the same
// convention as the other Must* helpers in this repo. Use it for cleanup
// that must not be silently skipped (deferred teardown, test setup).
func MustClose(closer Closer, opts ...Option) {
	if err := Close(closer, opts...); err != nil {
		panic(err)
	}
}
