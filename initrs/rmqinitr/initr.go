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
	config := &resolvedConfig{
		minRetryInterval: defaultMinRetryInterval,
		maxRetryInterval: defaultMaxRetryInterval,
		logger:           logr.Discard(),
	}
	if err := apply(config, opts); err != nil {
		return nil, err
	}

	if config.minRetryErr != nil {
		return nil, config.minRetryErr
	}
	if config.maxRetryErr != nil {
		return nil, config.maxRetryErr
	}
	if config.uri == "" {
		return nil, errors.New("rmqinitr: no rabbitmq configuration provided; pass WithConfig or WithURI")
	}
	if config.minRetryInterval <= 0 {
		return nil, fmt.Errorf("rmqinitr: min retry interval must be positive")
	}
	if config.maxRetryInterval < config.minRetryInterval {
		return nil, fmt.Errorf("rmqinitr: max retry interval (%v) is lower than min (%v)", config.maxRetryInterval, config.minRetryInterval)
	}

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	shell := &Shell{
		stopChan:     make(chan struct{}),
		config:       config,
		logger:       config.logger,
		workerCtx:    workerCtx,
		cancelWorker: cancelWorker,
		closeDone:    make(chan struct{}),
	}

	if !config.lazyConnect {
		conn, err := shell.tryConnect(ctx)
		if err != nil {
			cancelWorker()
			return nil, err
		}
		shell.conn = conn
		shell.setHealth(true)
	}

	shell.wg.Add(1)
	go shell.reconnectLoop()
	return shell, nil
}
