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
	store := &Store{
		Opts: &clientv3.Config{},
	}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	client, err := clientv3.New(*store.Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %w", err)
	}

	return &Shell{Client: client}, nil
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
