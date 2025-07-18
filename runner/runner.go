package runner

import (
	"context"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	name   string
	logger logr.Logger
	eg     *errgroup.Group
	ctx    context.Context
}

func New(ctx context.Context, name string, logger logr.Logger) *Runner {
	g, _ctx := errgroup.WithContext(ctx)
	return &Runner{eg: g, ctx: _ctx, logger: logger, name: name}
}

func (r *Runner) SetLimit(limit int) *Runner {
	r.eg.SetLimit(limit)
	return r
}

func (r *Runner) Add(runnable func() error) *Runner {
	r.eg.Go(runnable)
	return r
}

func (r *Runner) Run() error {
	err := r.eg.Wait()
	return err
}
