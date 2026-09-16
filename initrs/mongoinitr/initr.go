package mongoinitr

import (
	"context"
	"fmt"
	"time"

	"github.com/47monad/apin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Shell struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func MustNew(ctx context.Context, b apin.Builder[*Store]) *Shell {
	shell, err := _init(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	return _init(ctx, b)
}

func _init(ctx context.Context, b apin.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	client, err := mongo.Connect(store.Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err = client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("problem pinging database: %w", err)
	}

	shell := &Shell{Client: client}
	if store.DBName != "" {
		shell.DB = client.Database(store.DBName)
	}

	return shell, nil
}

func (shell *Shell) Close(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %w", err)
	}
	return nil
}
