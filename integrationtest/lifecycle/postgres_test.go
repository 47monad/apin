package lifecycle_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/47monad/apin"
	"github.com/47monad/apin/initrs/pginitr"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ apin.ReadinessChecker = (*pginitr.Shell)(nil)

func TestAppRunShutdownDeadlineWithPostgresPool(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
			servePostgresStartup(conn)
		}
	}()

	shell, err := pginitr.New(context.Background(),
		pginitr.WithURI("postgres://user:pass@"+listener.Addr().String()+"/test?sslmode=disable"),
		pginitr.WithNativePoolConfig(func(config *pgxpool.Config) error {
			config.MaxConns = 1
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("pginitr.New() error = %v", err)
	}
	defer func() { _ = shell.Close(context.Background()) }()

	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), time.Second)
	defer acquireCancel()
	acquired, err := shell.Pool.Acquire(acquireCtx)
	if err != nil {
		t.Fatalf("Pool.Acquire() error = %v", err)
	}

	serverConn := <-accepted
	t.Cleanup(func() { _ = serverConn.Close() })

	app := apin.New(apin.WithShutdownTimeout(100 * time.Millisecond))
	app.Track(shell)
	runCtx, cancelRun := context.WithCancel(context.Background())
	cancelRun()

	done := make(chan error, 1)
	started := time.Now()
	go func() { done <- app.Run(runCtx) }()

	select {
	case err := <-done:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("App.Run() error = %v, want shutdown deadline exceeded", err)
		}
	case <-time.After(time.Second):
		acquired.Release()
		<-done
		t.Fatalf("App.Run() exceeded shutdown deadline; elapsed %v", time.Since(started))
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("App.Run() took %v, want it bounded by the shutdown deadline", elapsed)
	}
	acquired.Release()
}

func servePostgresStartup(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	backend := pgproto3.NewBackend(conn, conn)
	if _, err := backend.ReceiveStartupMessage(); err != nil {
		return
	}
	backend.Send(&pgproto3.AuthenticationOk{})
	backend.Send(&pgproto3.ParameterStatus{Name: "server_version", Value: "16.0"})
	backend.Send(&pgproto3.ParameterStatus{Name: "client_encoding", Value: "UTF8"})
	backend.Send(&pgproto3.ParameterStatus{Name: "standard_conforming_strings", Value: "on"})
	backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
	if err := backend.Flush(); err != nil {
		return
	}
	_, _ = io.Copy(io.Discard, conn)
}
