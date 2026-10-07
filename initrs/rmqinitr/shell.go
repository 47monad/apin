package rmqinitr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-logr/logr"
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
	logger   logr.Logger

	workerCtx    context.Context
	cancelWorker context.CancelFunc
	dialConn     net.Conn
	closeOnce    sync.Once
	closeDone    chan struct{}

	config *resolvedConfig
}

func (r *Shell) reconnectLoop() {
	defer r.wg.Done()

	// A synchronous initial connection may already be established; monitor it
	// until it drops, then fall through to the reconnect loop.
	r.lock.RLock()
	initialConn, initialChan := r.conn, r.channel
	r.lock.RUnlock()
	if initialConn != nil {
		r.waitForClose(initialConn, initialChan)
		r.cleanupResources()
		r.setHealth(false)
		r.logger.Info("connection lost, attempting to reconnect...")
	}

	retryInterval := r.config.minRetryInterval

	for {
		select {
		case <-r.stopChan:
			return
		default:
			conn, ch, err := r.tryConnect(r.workerCtx)
			if err != nil {
				r.logger.Error(err, "rabbitmq reconnect failed")
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
			retryInterval = r.config.minRetryInterval

			r.lock.Lock()
			r.conn = conn
			r.channel = ch
			r.lock.Unlock()

			r.setHealth(true)
			r.logger.Info("rabbitmq connected successfully")

			// Wait for connection or channel to close
			r.waitForClose(conn, ch)

			// Clean up resources safely
			r.cleanupResources()
			r.setHealth(false)
			r.logger.Info("connection lost, attempting to reconnect...")
		}
	}
}

func (r *Shell) tryConnect(ctx context.Context) (*amqp.Connection, *amqp.Channel, error) {
	dialConfig := amqp.Config{Locale: "en_US"}
	if r.config.dialConfig != nil {
		dialConfig = *r.config.dialConfig
	}
	uri, err := amqp.ParseURI(r.config.uri)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse URI: %w", err)
	}
	connectionTimeout := 30 * time.Second
	if uri.ConnectionTimeout > 0 {
		connectionTimeout = time.Duration(uri.ConnectionTimeout) * time.Millisecond
	}
	nativeDial := dialConfig.Dial
	dialConfig.Dial = func(network, address string) (net.Conn, error) {
		var conn net.Conn
		if nativeDial != nil {
			conn, err = dialWithContext(ctx, connectionTimeout, nativeDial, network, address)
		} else {
			dialCtx, cancel := context.WithTimeout(ctx, connectionTimeout)
			conn, err = (&net.Dialer{}).DialContext(dialCtx, network, address)
			cancel()
		}
		if err != nil {
			return nil, err
		}
		if err := conn.SetDeadline(time.Now().Add(connectionTimeout)); err != nil {
			_ = conn.Close()
			return nil, err
		}
		if err := r.trackDialConnection(ctx, conn); err != nil {
			return nil, err
		}
		return conn, nil
	}
	defer r.clearDialConnection()

	conn, err := amqp.DialConfig(r.config.uri, dialConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("failed to create channel: %w", err)
	}

	return conn, ch, nil
}

// dialWithContext bounds a caller-provided native dial function, whose API
// does not accept a context. If it cannot be interrupted, its goroutine may
// finish later; any connection it eventually returns is then closed.
func dialWithContext(ctx context.Context, timeout time.Duration, dial func(string, string) (net.Conn, error), network, address string) (net.Conn, error) {
	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	type dialResult struct {
		conn net.Conn
		err  error
	}
	result := make(chan dialResult, 1)
	go func() {
		conn, err := dial(network, address)
		result <- dialResult{conn: conn, err: err}
	}()
	select {
	case result := <-result:
		return result.conn, result.err
	case <-dialCtx.Done():
		go func() {
			if result := <-result; result.conn != nil {
				_ = result.conn.Close()
			}
		}()
		return nil, dialCtx.Err()
	}
}

func (r *Shell) trackDialConnection(ctx context.Context, conn net.Conn) error {
	r.lock.Lock()
	if !r.closed && ctx.Err() == nil {
		r.dialConn = conn
		r.lock.Unlock()
		return nil
	}
	r.lock.Unlock()
	_ = conn.Close()
	if err := ctx.Err(); err != nil {
		return err
	}
	return ErrShellClosed
}

func (r *Shell) clearDialConnection() {
	r.lock.Lock()
	r.dialConn = nil
	r.lock.Unlock()
}

func (r *Shell) nextRetryInterval(current time.Duration) time.Duration {
	next := current * 2
	if next > r.config.maxRetryInterval {
		return r.config.maxRetryInterval
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
			r.logger.Error(fmt.Errorf("%v", err), "rabbitmq connection closed")
		}
	case err := <-chClosed:
		if err != nil {
			r.logger.Error(fmt.Errorf("%v", err), "rabbitmq channel closed")
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
			r.logger.Error(err, "error closing channel")
		}
		r.channel = nil
	}

	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			r.logger.Error(err, "error closing connection")
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

// Ready reports whether the RabbitMQ shell is currently healthy. It satisfies
// apin.ReadinessChecker and returns immediately; use WaitForHealth to wait for
// the connection to recover.
func (r *Shell) Ready(_ context.Context) error {
	r.lock.RLock()
	defer r.lock.RUnlock()
	if r.closed {
		return ErrShellClosed
	}
	if !r.healthy {
		return ErrNotHealthy
	}
	return nil
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
	r.closeOnce.Do(func() {
		r.lock.Lock()
		r.closed = true
		close(r.stopChan)
		dialConn := r.dialConn
		r.dialConn = nil
		cancelWorker := r.cancelWorker
		r.lock.Unlock()

		if cancelWorker != nil {
			cancelWorker()
		}
		if dialConn != nil {
			_ = dialConn.Close()
		}

		go func() {
			r.wg.Wait()
			r.cleanupResources()
			close(r.closeDone)
		}()
	})

	select {
	case <-r.closeDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
