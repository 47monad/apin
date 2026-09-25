package runner_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/47monad/apin/manifest"
	"github.com/47monad/apin/runner"
	"github.com/go-logr/logr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// freePort returns a port that is free right now. The runner binds it moments
// later, which is good enough for tests.
func freePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer lis.Close()
	return lis.Addr().(*net.TCPAddr).Port
}

// waitForListener blocks until the port accepts a connection.
func waitForListener(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr(port), 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("server on port %d never started listening", port)
}

func addr(port int) string {
	return "127.0.0.1:" + strconv.Itoa(port)
}

// runAsync starts Run in the background and returns a func that reports its
// result, failing the test if it has not returned by deadline.
func runAsync(t *testing.T, r *runner.Runner, timeout time.Duration) func() error {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run()
	}()
	return func() error {
		t.Helper()
		select {
		case err := <-errCh:
			return err
		case <-time.After(timeout):
			t.Fatalf("Run() did not return within %s", timeout)
			return nil
		}
	}
}

func TestHTTPServerGracefulShutdown(t *testing.T) {
	port := freePort(t)
	const work = 200 * time.Millisecond

	started := make(chan struct{})
	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddHTTPServer(&manifest.HTTPServerConfig{Port: port}, func(mux *http.ServeMux) {
		mux.HandleFunc("/slow", func(w http.ResponseWriter, req *http.Request) {
			close(started)
			time.Sleep(work)
			io.WriteString(w, "done")
		})
	})
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	type result struct {
		status int
		body   string
		err    error
	}
	resCh := make(chan result, 1)
	go func() {
		resp, err := http.Get("http://" + addr(port) + "/slow")
		if err != nil {
			resCh <- result{err: err}
			return
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		resCh <- result{status: resp.StatusCode, body: string(body), err: err}
	}()

	<-started
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}

	select {
	case res := <-resCh:
		if res.err != nil {
			t.Fatalf("in-flight request failed: %v", res.err)
		}
		if res.status != http.StatusOK {
			t.Errorf("in-flight request status = %d, want %d", res.status, http.StatusOK)
		}
		if res.body != "done" {
			t.Errorf("in-flight request body = %q, want %q", res.body, "done")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight request never completed")
	}

	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil after graceful shutdown", err)
	}
}

func TestGRPCServerGracefulStop(t *testing.T) {
	port := freePort(t)
	const work = 200 * time.Millisecond

	started := make(chan struct{})
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("api", healthpb.HealthCheckResponse_SERVING)

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		close(started)
		select {
		case <-time.After(work):
		case <-ctx.Done():
		}
		return handler(ctx, req)
	}))
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)

	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddGRPCServer(&manifest.GRPCServerConfig{Port: port}, grpcSrv)
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	conn, err := grpc.NewClient(addr(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()

	type result struct {
		resp *healthpb.HealthCheckResponse
		err  error
	}
	resCh := make(chan result, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		resp, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
		resCh <- result{resp: resp, err: err}
	}()

	<-started
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}

	select {
	case res := <-resCh:
		if res.err != nil {
			t.Fatalf("in-flight RPC failed instead of draining: %v", res.err)
		}
		if got := res.resp.GetStatus(); got != healthpb.HealthCheckResponse_SERVING {
			t.Errorf("in-flight RPC status = %v, want SERVING", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("in-flight RPC never completed")
	}

	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil after graceful stop", err)
	}
}

func TestStopReturnsErrorWhenDrainOutlivesContext(t *testing.T) {
	port := freePort(t)
	release := make(chan struct{})
	defer close(release)

	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddHTTPServer(&manifest.HTTPServerConfig{Port: port}, func(mux *http.ServeMux) {
		mux.HandleFunc("/hang", func(w http.ResponseWriter, req *http.Request) {
			<-release // never finishes within the test
		})
	})
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	go func() {
		resp, err := http.Get("http://" + addr(port) + "/hang")
		if err == nil {
			resp.Body.Close()
		}
	}()

	// Give the handler a moment to be in flight, then stop with a deadline
	// it cannot meet.
	time.Sleep(50 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Stop(ctx)
	if err == nil {
		t.Fatal("Stop() error = nil, want the drain deadline to be reported")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Stop() error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	// Stop still cancelled the group, so Run returns.
	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestStopHardStopsGRPCWhenDrainOutlivesContext(t *testing.T) {
	port := freePort(t)

	started := make(chan struct{}, 1)
	healthSrv := health.NewServer()
	healthSrv.SetServingStatus("api", healthpb.HealthCheckResponse_SERVING)

	release := make(chan struct{})
	defer close(release)

	grpcSrv := grpc.NewServer(grpc.UnaryInterceptor(func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release: // outlives the Stop deadline
		case <-ctx.Done():
		}
		return handler(ctx, req)
	}))
	healthpb.RegisterHealthServer(grpcSrv, healthSrv)

	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddGRPCServer(&manifest.GRPCServerConfig{Port: port}, grpcSrv)
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	conn, err := grpc.NewClient(addr(port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer conn.Close()

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{})
	}()

	<-started
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err = r.Stop(ctx)
	if err == nil {
		t.Fatal("Stop() error = nil, want the drain deadline to be reported")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Stop() error = %v, want it to wrap context.DeadlineExceeded", err)
	}

	// The hard stop must let Run return rather than hang on GracefulStop.
	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestStopIsIdempotent(t *testing.T) {
	port := freePort(t)
	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddHTTPServer(&manifest.HTTPServerConfig{Port: port}, nil)
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	var wg sync.WaitGroup
	errs := make([]error, 4)
	for i := range errs {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = r.Stop(context.Background())
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("concurrent Stop() #%d error = %v, want nil", i, err)
		}
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Errorf("Stop() after shutdown error = %v, want nil", err)
	}
	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestStopCancelsContext(t *testing.T) {
	r := runner.New(context.Background(), "api", logr.Discard())
	if err := r.Context().Err(); err != nil {
		t.Fatalf("Context() is already done before shutdown: %v", err)
	}

	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v, want nil", err)
	}

	select {
	case <-r.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("Context() was not cancelled by Stop")
	}
}

func TestContextFollowsParent(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	r := runner.New(parent, "api", logr.Discard())

	cancel()
	select {
	case <-r.Context().Done():
	case <-time.After(time.Second):
		t.Fatal("Context() did not follow the parent context")
	}
}

func TestRunDrainsServersOnParentCancel(t *testing.T) {
	port := freePort(t)
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := runner.New(parent, "api", logr.Discard())
	r.AddHTTPServer(&manifest.HTTPServerConfig{Port: port}, nil)
	r.AddHealthCheck(health.NewServer(), 10*time.Millisecond, func(context.Context) bool { return true })
	runResult := runAsync(t, r, 5*time.Second)
	waitForListener(t, port)

	// Cancel the parent: Run must drain the server and the health checker
	// instead of blocking forever.
	cancel()
	if err := runResult(); err != nil {
		t.Errorf("Run() error = %v, want nil after parent cancellation", err)
	}

	// The port must be free again.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr(port), 100*time.Millisecond)
		if err != nil {
			return
		}
		conn.Close()
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("http server is still accepting connections after Run returned")
}

func TestRunReturnsRunnableError(t *testing.T) {
	want := errors.New("runnable failed")
	r := runner.New(context.Background(), "api", logr.Discard())
	r.Add(func() error { return want })

	if err := r.Run(); !errors.Is(err, want) {
		t.Errorf("Run() error = %v, want %v", err, want)
	}
}

func TestAddRecoversPanic(t *testing.T) {
	r := runner.New(context.Background(), "api", logr.Discard())
	r.Add(func() error { panic("boom") })

	err := r.Run()
	if err == nil {
		t.Fatal("Run() error = nil, want a panic-derived error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Run() error = %v, want it to mention the panic value", err)
	}
}

func TestAddIgnoresNilRunnable(t *testing.T) {
	r := runner.New(context.Background(), "api", logr.Discard())
	r.Add(nil).Add(func() error { return nil })

	if err := r.Run(); err != nil {
		t.Errorf("Run() error = %v, want nil", err)
	}
}

func TestSetLimitZeroDoesNotDeadlock(t *testing.T) {
	r := runner.New(context.Background(), "api", logr.Discard())

	started := make(chan struct{})
	r.SetLimit(0).Add(func() error {
		close(started)
		return nil
	})

	errCh := make(chan error, 1)
	go func() { errCh <- r.Run() }()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("runnable never started: SetLimit(0) deadlocked the group")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return")
	}
}

func TestHTTPServerListenErrorIsReported(t *testing.T) {
	// Hold the port so the runner cannot bind it.
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer lis.Close()
	port := lis.Addr().(*net.TCPAddr).Port

	r := runner.New(context.Background(), "api", logr.Discard())
	r.AddHTTPServer(&manifest.HTTPServerConfig{Port: port}, nil)

	if err := r.Run(); err == nil {
		t.Error("Run() error = nil, want the listen failure")
	}
}
