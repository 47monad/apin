package rmqinitr

import (
	"context"

	"github.com/47monad/apin/initropts"
)

func MustNew(ctx context.Context, b initropts.Builder[*Store]) *Shell {
	shell, err := _init(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	mgr := &Shell{
		stopChan: make(chan struct{}),
		healthy:  false,
		closed:   false,
		store:    store,
	}

	mgr.wg.Add(1)
	go mgr.reconnectLoop()
	return mgr, nil
}
