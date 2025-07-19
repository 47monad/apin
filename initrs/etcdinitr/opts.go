package etcdinitr

import (
	"strings"
	"time"

	"github.com/47monad/zaal"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type Store struct {
	Opts *clientv3.Config
}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) WithConfig(config *zaal.EtcdConfig) *Builder {
	b.SetEndpoints(strings.Split(config.Endpoints, ","))
	if config.Username != "" {
		b.SetUsername(config.Username)
	}
	if config.Password != "" {
		b.SetPassword(config.Password)
	}
	if config.Timeout > 0 {
		b.SetTimeout(time.Duration(config.Timeout) * time.Second)
	}
	return b
}

func (b *Builder) Build() (*Store, error) {
	store := &Store{
		Opts: &clientv3.Config{},
	}

	for _, opt := range b.Opts {
		if opt == nil {
			continue
		}

		if err := opt(store); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (b *Builder) SetEndpoints(endpoints []string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.Endpoints = endpoints
		return nil
	})
	return b
}

func (b *Builder) SetUsername(username string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.Username = username
		return nil
	})
	return b
}

func (b *Builder) SetPassword(password string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.Password = password
		return nil
	})
	return b
}

func (b *Builder) SetTimeout(d time.Duration) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.DialTimeout = d
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
