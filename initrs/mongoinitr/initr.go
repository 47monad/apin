package mongoinitr

import (
	"context"
	"fmt"
	"time"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Shell struct {
	Client *mongo.Client
	Db     *mongo.Database
}

var Mongodb = initr.AgentFunc[*Store, *Shell](initMongodb)

func EnsureMongodb(ctx context.Context, b initropts.Builder[*Store]) *Shell {
	shell, err := initMongodb(ctx, b)
	if err != nil {
		panic(err)
	}
	return shell
}

func initMongodb(ctx context.Context, b initropts.Builder[*Store]) (*Shell, error) {
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

	return &Shell{Client: client}, nil
}

func (shell *Shell) Shutdown(ctx context.Context) error {
	if shell.Client == nil {
		return nil
	}

	err := shell.Client.Disconnect(ctx)
	if err != nil {
		return fmt.Errorf("failed to disconnect from mongodb: %v", err)
	}
	return nil
}
