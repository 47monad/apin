package initr

import (
	"context"
	"fmt"

	"github.com/47monad/apin/initropts"
	"github.com/rabbitmq/amqp091-go"
)

type RabbitMQShell struct {
	Conn    *amqp091.Connection
	Channel *amqp091.Channel
}

var RabbitMQ = AgentFunc[*initropts.RabbitMQStore, *RabbitMQShell](initRabbitMQ)

func initRabbitMQ(_ context.Context, b initropts.Builder[*initropts.RabbitMQStore]) (*RabbitMQShell, error) {
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

	return &RabbitMQShell{Conn: conn, Channel: ch}, nil
}

func (shell *RabbitMQShell) Dispose(ctx context.Context) error {
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
