package rmqinitr

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
)

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store := &Store{
		MinRetryInterval: defaultMinRetryInterval,
		MaxRetryInterval: defaultMaxRetryInterval,
		Logger:           logr.Discard(),
	}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	if store.URI == "" {
		return nil, errors.New("rmqinitr: no rabbitmq configuration provided; pass WithConfig or WithURI")
	}
	if store.MaxRetryInterval < store.MinRetryInterval {
		return nil, fmt.Errorf("rmqinitr: max retry interval (%v) is lower than min (%v)", store.MaxRetryInterval, store.MinRetryInterval)
	}

	shell := &Shell{
		stopChan: make(chan struct{}),
		store:    store,
		logger:   store.Logger,
	}

	if !store.LazyConnect {
		conn, ch, err := shell.tryConnect()
		if err != nil {
			return nil, err
		}
		shell.conn = conn
		shell.channel = ch
		shell.setHealth(true)
	}

	shell.wg.Add(1)
	go shell.reconnectLoop()
	return shell, nil
}
