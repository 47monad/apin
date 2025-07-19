package etcdinitr

import (
	"context"
	"fmt"

	"github.com/47monad/apin"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Shell struct {
	Client *clientv3.Client
}

func New(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func MustNew(ctx context.Context, b apin.Builder[*Store]) *Shell {
	shell, err := New(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func _init(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	client, err := clientv3.New(*store.Opts)
	if err != nil {
		panic(err)
	}

	return &Shell{Client: client}, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Close()
	if err != nil {
		return fmt.Errorf("failed to disconnect from etcd: %v", err)
	}
	return nil
}
