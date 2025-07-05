package mongoinitr

import (
	"context"
	"fmt"
	"time"

	"github.com/47monad/apin/initropts"
	"github.com/47monad/zaal"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Shell struct {
	Client *mongo.Client
	Db     *mongo.Database
}

func MustNewFromConfig(ctx context.Context, config *zaal.MongodbConfig) *Shell {
	shell, err := NewFromConfig(ctx, config)
	if err != nil {
		panic(err)
	}
	return shell
}

func NewFromConfig(ctx context.Context, config *zaal.MongodbConfig) (*Shell, error) {
	b := Opts()
	b.SetURI(config.URI)
	if config.DbName != "" {
		b.SetDBName(config.DbName)
	}
	return _init(ctx, b)
}

func MustNew(ctx context.Context, b initropts.Builder[*Store]) *Shell {
	shell, err := _init(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	client, err := mongo.Connect(store.Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("problem pinging database: %v", err)
	}

	shell := &Shell{Client: client}
	if store.DBName != "" {
		shell.Db = client.Database(store.DBName)
	}

	return shell, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %v", err)
	}
	return nil
}
