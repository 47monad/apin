package mongoinitr

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store struct {
	Opts   *options.ClientOptions
	DBName string
}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	store := &Store{
		Opts: options.Client(),
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

func (b *Builder) SetURI(uri string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.ApplyURI(uri)
		return nil
	})
	return b
}

func (b *Builder) SetTimeout(d time.Duration) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.Opts.SetConnectTimeout(d)
		return nil
	})
	return b
}

func (b *Builder) SetDBName(name string) *Builder {
	b.Opts = append(b.Opts, func(o *Store) error {
		o.DBName = name
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
