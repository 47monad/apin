package zapinitr

type Store struct{}

type Builder struct {
	Opts []func(*Store) error
}

func (b *Builder) Build() (*Store, error) {
	return &Store{}, nil
}

func Opts() *Builder {
	return &Builder{}
}
