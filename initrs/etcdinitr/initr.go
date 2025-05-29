package etcdinitr

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/47monad/apin/initropts"
	"github.com/47monad/zaal"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Shell struct {
	Client *clientv3.Client
}

func New(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func MustNew(ctx context.Context, b initropts.Builder[*Store]) *Shell {
	shell, err := New(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func NewFromConfig(ctx context.Context, config *zaal.EtcdConfig) (*Shell, error) {
	o := Opts()
	o.SetEndpoints(strings.Split(config.Endpoints, ","))
	if config.Username != "" {
		o.SetUsername(config.Username)
	}
	if config.Password != "" {
		o.SetPassword(config.Password)
	}
	if config.Timeout > 0 {
		o.SetTimeout(time.Duration(config.Timeout) * time.Second)
	}
	return _init(ctx, o)
}

func MustNewFromConfig(ctx context.Context, config *zaal.EtcdConfig) *Shell {
	shell, err := NewFromConfig(ctx, config)
	if err != nil {
		panic(err)
	}
	return shell
}

func _init(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
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

func (shell *Shell) Shutdown(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Close()
	if err != nil {
		return fmt.Errorf("failed to disconnect from etcd: %v", err)
	}
	return nil
}
