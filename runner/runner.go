package runner

import (
	"context"
	"github.com/47monad/apin"
	"golang.org/x/sync/errgroup"
)

type Runner struct {
	app *apin.App
	eg  *errgroup.Group
	ctx context.Context
}

func New(ctx context.Context, app *apin.App, limit int) *Runner {
	g, _ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	return &Runner{eg: g, ctx: _ctx, app: app}
}

func (r *Runner) Add(runnable func() error) {
	r.eg.Go(runnable)
}

func (r *Runner) Run() error {
	err := r.eg.Wait()
	return err
}
