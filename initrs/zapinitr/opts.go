package zapinitr

import "github.com/47monad/zaal"

type Store struct{}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	return &Store{}, nil
}

func (b *Builder) WithConfig(config *zaal.LoggingConfig) *Builder {
	return b
}

func Opts() *Builder {
	return &Builder{}
}
