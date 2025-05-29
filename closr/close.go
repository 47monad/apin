package closr

import "context"

type Closer interface {
	Close(ctx context.Context) error
}

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

func MustClose(closer Closer, opts ...Option) {
	_ = Close(closer, opts...)
}
