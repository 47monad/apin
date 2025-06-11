package rmqinitr

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	ErrShellClosed = errors.New("rabbitmq shell is closed")
	ErrNotHealthy  = errors.New("rabbitmq shell is not healthy")
)

type Shell struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	lock     sync.RWMutex
	healthy  bool
	closed   bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	store *Store
}

func (r *Shell) reconnectLoop() {
	defer r.wg.Done()

	retryInterval := r.store.MinRetryInterval

	for {
		select {
		case <-r.stopChan:
			return
		default:
			conn, ch, err := r.tryConnect()
			if err != nil {
				log.Printf("RabbitMQ reconnect failed: %v", err)
				r.setHealth(false)

				// Exponential backoff
				select {
				case <-r.stopChan:
					return
				case <-time.After(retryInterval):
					retryInterval = r.nextRetryInterval(retryInterval)
				}
				continue
			}

			// Reset retry interval on successful connection
			retryInterval = r.store.MinRetryInterval

			r.lock.Lock()
			r.conn = conn
			r.channel = ch
			r.lock.Unlock()

			r.setHealth(true)
			log.Println("RabbitMQ connected successfully")

			// Wait for connection or channel to close
			r.waitForClose(conn, ch)

			// Clean up resources safely
			r.cleanupResources()
			r.setHealth(false)
			log.Println("Connection lost, attempting to reconnect...")
		}
	}
}

func (r *Shell) tryConnect() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(r.store.URI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("failed to create channel: %w", err)
	}

	return conn, ch, nil
}

func (r *Shell) nextRetryInterval(current time.Duration) time.Duration {
	next := current * 2
	if next > r.store.MaxRetryInterval {
		return r.store.MaxRetryInterval
	}
	return next
}

func (r *Shell) waitForClose(conn *amqp.Connection, ch *amqp.Channel) {
	connClosed := make(chan *amqp.Error, 1)
	chClosed := make(chan *amqp.Error, 1)

	conn.NotifyClose(connClosed)
	ch.NotifyClose(chClosed)

	select {
	case err := <-connClosed:
		if err != nil {
			log.Printf("RabbitMQ connection closed: %v", err)
		}
	case err := <-chClosed:
		if err != nil {
			log.Printf("RabbitMQ channel closed: %v", err)
		}
	case <-r.stopChan:
		return
	}
}

func (r *Shell) cleanupResources() {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.channel != nil {
		if err := r.channel.Close(); err != nil {
			log.Printf("Error closing channel: %v", err)
		}
		r.channel = nil
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			log.Printf("Error closing connection: %v", err)
		}
		r.conn = nil
	}
}

func (r *Shell) setHealth(h bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.healthy = h
}

func (r *Shell) IsHealthy() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()
	return r.healthy && !r.closed
}

func (r *Shell) GetChannel() (*amqp.Channel, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if r.closed {
		return nil, ErrShellClosed
	}

	if !r.healthy || r.channel == nil {
		return nil, ErrNotHealthy
	}

	return r.channel, nil
}

func (r *Shell) GetConn() (*amqp.Connection, error) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	if r.closed {
		return nil, ErrShellClosed
	}

	if !r.healthy || r.conn == nil {
		return nil, ErrNotHealthy
	}

	return r.conn, nil
}

func (r *Shell) WaitForHealth(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if r.IsHealthy() {
				return nil
			}
		}
	}
}

func (r *Shell) Close(ctx context.Context) error {
	r.lock.Lock()
	if r.closed {
		r.lock.Unlock()
		return nil
	}
	r.closed = true
	r.lock.Unlock()

	close(r.stopChan)
	r.wg.Wait()

	r.cleanupResources()
	return nil
}
