package lifecycle_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/47monad/apin"
	"github.com/47monad/apin/initrs/rmqinitr"
)

func TestAppRunShutdownDeadlineWithRabbitMQReconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	accepted := make(chan net.Conn, 1)
	go func() {
		firstConn, err := listener.Accept()
		if err != nil {
			return
		}
		_ = firstConn.Close() // Force the worker to enter its reconnect cycle.

		reconnectConn, err := listener.Accept()
		if err == nil {
			accepted <- reconnectConn
		}
	}()

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithURI("amqp://guest:guest@"+listener.Addr().String()+"/"),
		rmqinitr.WithLazyConnect(),
		rmqinitr.WithMinRetryInterval(10*time.Millisecond),
		rmqinitr.WithMaxRetryInterval(20*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("rmqinitr.New() error = %v", err)
	}
	defer shell.Close(context.Background())

	var serverConn net.Conn
	select {
	case serverConn = <-accepted:
	case <-time.After(time.Second):
		t.Fatal("RabbitMQ shell did not begin its reconnect attempt")
	}
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
		if err != nil {
			t.Errorf("App.Run() error = %v, want clean shutdown", err)
		}
	case <-time.After(time.Second):
		_ = serverConn.Close()
		<-done
		t.Fatalf("App.Run() exceeded shutdown deadline; elapsed %v", time.Since(started))
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Errorf("App.Run() took %v, want it bounded by the shutdown deadline", elapsed)
	}
}
