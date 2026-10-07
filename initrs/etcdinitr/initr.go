package etcdinitr

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// Ready checks whether any configured etcd endpoint responds to a status
// request. Callers can bound the probe by passing a context with a deadline.
func (shell *Shell) Ready(ctx context.Context) error {
	if shell.Client == nil {
		return errors.New("etcdinitr: shell is not initialized")
	}

	endpoints := shell.Client.Endpoints()
	if len(endpoints) == 0 {
		return errors.New("etcdinitr: no endpoints configured")
	}

	var errs []error
	for _, endpoint := range endpoints {
		if _, err := shell.Client.Status(ctx, endpoint); err == nil {
			return nil
		} else {
			errs = append(errs, fmt.Errorf("endpoint %q: %w", endpoint, err))
		}
		if ctx.Err() != nil {
			break
		}
	}

	return fmt.Errorf("etcdinitr: no configured endpoint is ready: %w", errors.Join(errs...))
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
	if len(config.opts.Endpoints) == 0 {
		return nil, fmt.Errorf("etcd endpoints must not be empty")
	}
	for i, endpoint := range config.opts.Endpoints {
		if strings.TrimSpace(endpoint) == "" {
			return nil, fmt.Errorf("etcd endpoint %d must not be blank", i)
		}
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
