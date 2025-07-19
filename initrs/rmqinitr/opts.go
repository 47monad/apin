package rmqinitr

import (
	"time"

	"github.com/47monad/zaal"
)

type Store struct {
	URI              string
	MaxRetryInterval time.Duration
	MinRetryInterval time.Duration
}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	store := &Store{}

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

func (b *Builder) WithConfig(config *zaal.RabbitMQConfig) *Builder {
	b.SetURI(config.URI)
	if config.MinRetryInterval == 0 {
		config.MinRetryInterval = 1
	}
	if config.MaxRetryInterval == 0 {
		config.MaxRetryInterval = 30
	}
	b.SetMinRetryInterval(time.Duration(config.MinRetryInterval) * time.Second)
	b.SetMaxRetryInterval(time.Duration(config.MaxRetryInterval) * time.Second)
	return b
}

func (b *Builder) SetURI(uri string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.URI = uri
		return nil
	})
	return b
}

func (b *Builder) SetMinRetryInterval(interval time.Duration) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.MinRetryInterval = interval
		return nil
	})
	return b
}

func (b *Builder) SetMaxRetryInterval(interval time.Duration) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.MaxRetryInterval = interval
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
