package httpinitr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
)

const defaultPort = 4747

// ServerShell exposes the native HTTP server and context-aware serving.
type ServerShell struct {
	Server *http.Server

	mu        sync.Mutex
	listeners []net.Listener
	closed    bool
}

// Listen opens and records a listener at Server.Addr. The shell closes it
// during Close even if Serve has not started yet.
func (shell *ServerShell) Listen() (net.Listener, error) {
	if shell == nil || shell.Server == nil {
		return nil, errors.New("httpinitr: cannot listen with a nil server")
	}
	shell.mu.Lock()
	defer shell.mu.Unlock()
	if shell.closed {
		return nil, errors.New("httpinitr: cannot listen after close")
	}
	listener, err := net.Listen("tcp", shell.Server.Addr)
	if err != nil {
		return nil, err
	}
	shell.listeners = append(shell.listeners, listener)
	return listener, nil
}

// MustNew is New but panics on failure.
func MustNew(ctx context.Context, opts ...Option) *ServerShell {
	shell, err := New(ctx, opts...)
	if err != nil {
		panic(err)
	}
	return shell
}

// New constructs an HTTP server without binding its port. Applications can
// configure the server further through Server before serving it.
func New(_ context.Context, opts ...Option) (*ServerShell, error) {
	store := &Store{Port: defaultPort}
	if err := apply(store, opts); err != nil {
		return nil, err
	}
	if store.Port == 0 {
		store.Port = defaultPort
	}
	if store.Port < 1 || store.Port > 65535 {
		return nil, fmt.Errorf("httpinitr: port must be between 1 and 65535, got %d", store.Port)
	}
	if store.Handler == nil {
		store.Handler = http.NewServeMux()
	}

	server := &http.Server{
		Addr:    net.JoinHostPort("", strconv.Itoa(store.Port)),
		Handler: store.Handler,
	}
	return &ServerShell{Server: server}, nil
}

// Serve serves on listener until the context ends or the server fails. The
// caller should track the shell with App so shutdown closes the native server.
func (shell *ServerShell) Serve(ctx context.Context, listener net.Listener) error {
	if shell == nil || shell.Server == nil {
		return errors.New("httpinitr: cannot serve with a nil server")
	}
	if listener == nil {
		return errors.New("httpinitr: cannot serve with a nil listener")
	}
	if err := shell.trackListener(listener); err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() { errCh <- shell.Server.Serve(listener) }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return nil
	}
}

// Close gracefully shuts down the native HTTP server using ctx. If graceful
// shutdown exceeds its deadline, it force-closes active connections.
func (shell *ServerShell) Close(ctx context.Context) error {
	if shell == nil || shell.Server == nil {
		return nil
	}
	shutdownErr := shell.Server.Shutdown(ctx)
	var closeErr error
	if shutdownErr != nil {
		closeErr = shell.Server.Close()
	}
	return errors.Join(shutdownErr, closeErr, shell.closeListeners())
}

func (shell *ServerShell) trackListener(listener net.Listener) error {
	shell.mu.Lock()
	defer shell.mu.Unlock()
	if shell.closed {
		return errors.New("httpinitr: cannot serve after close")
	}
	shell.listeners = append(shell.listeners, listener)
	return nil
}

func (shell *ServerShell) closeListeners() error {
	shell.mu.Lock()
	shell.closed = true
	listeners := shell.listeners
	shell.listeners = nil
	shell.mu.Unlock()

	var errs []error
	for _, listener := range listeners {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
