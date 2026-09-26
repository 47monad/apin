package etcdinitr

import (
	"context"
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type Shell struct {
	Client *clientv3.Client
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	config, err := resolveConfig(opts)
	if err != nil {
		return nil, err
	}

	client, err := clientv3.New(*config.opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return &Shell{Client: client}, nil
}

func resolveConfig(opts []Option) (*resolvedConfig, error) {
	config := &resolvedConfig{opts: &clientv3.Config{}}
	if err := apply(config, opts); err != nil {
		return nil, err
	}
	if config.timeoutErr != nil {
		return nil, config.timeoutErr
	}
	if config.opts.DialTimeout < 0 {
		return nil, fmt.Errorf("etcd dial timeout must not be negative")
	}
	return config, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Close()
	if err != nil {
		return fmt.Errorf("failed to disconnect from etcd: %w", err)
	}
	return nil
}
