package prominitr

import "github.com/47monad/zaal"

type Store struct{}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	return &Store{}, nil
}

func (b *Builder) WithConfig(config *zaal.PrometheusConfig) *Builder {
	// TODO: enable grpc metrics here
	return b
}

func Opts() *Builder {
	return &Builder{}
}
