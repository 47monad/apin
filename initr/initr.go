package initr

import (
	"context"
	"github.com/47monad/apin/initropts"
)

type Agent[K any, T any] interface {
	Prepare(ctx context.Context, builder initropts.Builder[K]) (T, error)
}

type AgentFunc[K any, T any] func(context.Context, initropts.Builder[K]) (T, error)

func (f AgentFunc[K, T]) Prepare(ctx context.Context, b initropts.Builder[K]) (T, error) {
	return f(ctx, b)
}

type Disposer interface {
	Dispose(ctx context.Context) error
}

type DisposerOptions struct {
	ctx context.Context
}

type DisposerOption func(o *DisposerOptions)

func WithContext(ctx context.Context) DisposerOption {
	return func(o *DisposerOptions) {
		o.ctx = ctx
	}
}

func Dispose(disposer Disposer, opts ...DisposerOption) error {
	_opts := &DisposerOptions{
		ctx: context.Background(),
	}
	for _, opt := range opts {
		opt(_opts)
	}
	return disposer.Dispose(_opts.ctx)
}

func EnsureDisposed(disposer Disposer, opts ...DisposerOption) {
	_ = Dispose(disposer, opts...)
}
