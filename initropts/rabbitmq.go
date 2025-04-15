package initropts

type RabbitMQStore struct {
	URI string
}

type RabbitMQBuilder struct {
	Opts []func(*RabbitMQStore) error
}

func (b *RabbitMQBuilder) Build() (*RabbitMQStore, error) {
	store := &RabbitMQStore{}

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

func (b *RabbitMQBuilder) SetUri(uri string) *RabbitMQBuilder {
	b.Opts = append(b.Opts, func(o *RabbitMQStore) error {
		o.URI = uri
		return nil
	})
	return b
}

func RabbitMQ() *RabbitMQBuilder {
	return &RabbitMQBuilder{}
}
