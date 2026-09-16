package mongoinitr

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Shell struct {
	Client *mongo.Client
	DB     *mongo.Database
}

func MustNew(ctx context.Context, opts ...Option) *Shell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

func New(ctx context.Context, opts ...Option) (*Shell, error) {
	store := &Store{
		Opts:        options.Client(),
		PingTimeout: 10 * time.Second,
	}
	if err := apply(store, opts); err != nil {
		return nil, err
	}

	client, err := mongo.Connect(store.Opts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, store.PingTimeout)
	defer cancel()

	if err = client.Ping(pingCtx, nil); err != nil {
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
