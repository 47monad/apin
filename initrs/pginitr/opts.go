package pginitr

import "github.com/47monad/zaal"

type Store struct {
	URI string
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

func (b *Builder) WithConfig(config *zaal.PostgresConfig) *Builder {
	b.SetURI(config.URI)
	return b
}

func (b *Builder) SetURI(uri string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI = uri
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
