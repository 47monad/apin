package initropts

type ZapLoggerStore struct {
}

type ZapLoggerBuilder struct {
	Opts []func(*ZapLoggerStore) error
}

func (b *ZapLoggerBuilder) Build() (*ZapLoggerStore, error) {
	return &ZapLoggerStore{}, nil
}

func Zap() *ZapLoggerBuilder {
	return &ZapLoggerBuilder{}
}
