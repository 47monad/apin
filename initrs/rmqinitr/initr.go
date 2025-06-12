package rmqinitr

import (
	"context"
	"time"

	"github.com/47monad/apin/initropts"
	"github.com/47monad/zaal"
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

func MustNewFromConfig(ctx context.Context, config *zaal.RabbitMQConfig) *Shell {
	shell, err := NewFromConfig(ctx, config)
	if err != nil {
		panic(err)
	}
	return shell
}

func NewFromConfig(ctx context.Context, config *zaal.RabbitMQConfig) (*Shell, error) {
	b := Opts()
	b.SetURI(config.URI)
	if config.MinRetryInterval == 0 {
		config.MinRetryInterval = 1
	}
	if config.MaxRetryInterval == 0 {
		config.MaxRetryInterval = 30
	}
	b.SetMinRetryInterval(time.Duration(config.MinRetryInterval) * time.Second)
	b.SetMaxRetryInterval(time.Duration(config.MaxRetryInterval) * time.Second)
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
