package rmqinitr

import "time"

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
