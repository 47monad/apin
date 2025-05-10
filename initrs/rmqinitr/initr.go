package rmqinitr

import (
	"context"
	"fmt"

	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/initropts"
	"github.com/rabbitmq/amqp091-go"
)

type Shell struct {
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

var RabbitMQ = initr.AgentFunc[*Store, *Shell](initRabbitMQ)

func initRabbitMQ(_ context.Context, b initropts.Builder[*Store]) (*Shell, error) {
	store, err := b.Build()
	if err != nil {
		return nil, err
	}

	conn, err := amqp091.Dial(store.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq server: %v", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to create channel: %v", err)
	}

	return &Shell{Conn: conn, Channel: ch}, nil
}

func (shell *Shell) Dispose(ctx context.Context) error {
	if shell.Channel != nil {
		err := shell.Channel.Close()
		if err != nil {
			return fmt.Errorf("failed to close channel")
		}
	}

	if shell.Conn == nil {
		return nil
	}
	err := shell.Conn.Close()
	if err != nil {
		return fmt.Errorf("failed to close connection: %v", err)
	}
	return nil
}
