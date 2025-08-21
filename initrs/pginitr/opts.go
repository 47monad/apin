package pginitr

import (
	"net/url"

	"github.com/47monad/zaal"
)

// TODO: SSLMode and Params are not being used anywhere

type Store struct {
	URI *url.URL
}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	store := &Store{
		URI: &url.URL{Scheme: "postgres"},
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

func (b *Builder) WithConfig(config *zaal.PostgresConfig) *Builder {
	if config.URI != "" {
		b.ApplyURI(config.URI)
	}
	if config.Username != "" {
		b.SetUser(url.UserPassword(config.Username, config.Password))
	}
	if config.Host != "" {
		b.SetHost(config.Host)
	}
	if config.DBName != "" {
		b.SetDBName(config.DBName)
	}
	return b
}

func (b *Builder) ApplyURI(uri string) *Builder {
	// TODO: handle the error
	parsed, _ := ParseURI(uri)
	if parsed.User.Username() != "" {
		b.SetUser(parsed.User)
	}
	if parsed.Path != "" {
		b.SetDBName(parsed.Path)
	}
	if parsed.Host != "" {
		b.SetHost(parsed.Host)
	}
	return b
}

func (b *Builder) SetUser(user *url.Userinfo) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.User = user
		return nil
	})
	return b
}

func (b *Builder) SetHost(host string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.Host = host
		return nil
	})
	return b
}

func (b *Builder) SetDBName(dbname string) *Builder {
	b.Opts = append(b.Opts, func(s *Store) error {
		s.URI.Path = dbname
		return nil
	})
	return b
}

func Opts() *Builder {
	return &Builder{}
}
