package initropts

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongodbStore struct {
	Opts *options.ClientOptionsBuilder
}

type MongodbBuilder struct {
	Opts []func(*MongodbStore) error
}

func (b *MongodbBuilder) Build() (*MongodbStore, error) {
	store := &MongodbStore{
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

func (b *MongodbBuilder) SetUri(uri string) *MongodbBuilder {
	b.Opts = append(b.Opts, func(o *MongodbStore) error {
		o.Opts.ApplyURI(uri)
		return nil
	})
	return b
}

func (b *MongodbBuilder) SetTimeout(d time.Duration) *MongodbBuilder {
	b.Opts = append(b.Opts, func(o *MongodbStore) error {
		o.Opts.SetConnectTimeout(d)
		return nil
	})
	return b
}

func Mongodb() *MongodbBuilder {
	return &MongodbBuilder{}
}
