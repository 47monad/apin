package integrationtest

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/rmqinitr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	rmqtc "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

const rabbitmqImage = "rabbitmq:4.1.1-management-alpine"

var (
	rabbitmqURI string
	container   *rmqtc.RabbitMQContainer

	// dockerUnavailable is set by TestMain when the Docker provider cannot be
	// reached. Tests that need the shared broker then skip with this reason.
	dockerUnavailable error
)

func seconds(value int) *int { return &value }

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	ctx := context.Background()

	provider, err := testcontainers.ProviderDocker.GetProvider()
	if err != nil {
		dockerUnavailable = fmt.Errorf("docker provider unavailable: %w", err)
		return m.Run()
	}
	defer func() { _ = provider.Close() }()

	if err := provider.Health(ctx); err != nil {
		dockerUnavailable = fmt.Errorf("docker is not healthy: %w", err)
		return m.Run()
	}

	container, err = rmqtc.Run(ctx, rabbitmqImage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not start rabbitmq container: %v\n", err)
		return 1
	}
	defer func() {
		if err := container.Terminate(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "could not terminate rabbitmq container: %v\n", err)
		}
	}()

	// AmqpURL resolves the mapped container port for 5672/tcp.
	rabbitmqURI, err = container.AmqpURL(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not get amqp URI: %v\n", err)
		return 1
	}

	return m.Run()
}

// requireDocker skips tests that depend on the shared broker when Docker is
// unavailable, reporting why.
func requireDocker(t *testing.T) {
	t.Helper()
	if dockerUnavailable != nil {
		t.Skipf("skipping rabbitmq integration test: %v", dockerUnavailable)
	}
}

func TestRequireDockerSkipsWhenUnavailable(t *testing.T) {
	restore := dockerUnavailable
	dockerUnavailable = fmt.Errorf("simulated docker outage")
	defer func() { dockerUnavailable = restore }()

	ran := false
	t.Run("docker dependent test", func(t *testing.T) {
		requireDocker(t)
		ran = true
	})
	if ran {
		t.Fatal("requireDocker should have skipped a docker-dependent test")
	}
}

func TestNewFromConfig_ValidConnection(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{URI: rabbitmqURI}))
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	assert.True(t, shell.IsHealthy())
}

func TestNewRabbitManager_InvalidConnection(t *testing.T) {
	// Synchronous connect must fail on an invalid connection.
	_, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: "amqp://invalid:invalid@localhost:9999/",
		}))
	require.Error(t, err)
}

func TestNewRabbitManager_InvalidConnectionLazy(t *testing.T) {
	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: "amqp://invalid:invalid@localhost:9999/",
		}),
		rmqinitr.WithLazyConnect())
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	// A lazy shell must not become healthy against an unreachable broker.
	assert.Never(t, shell.IsHealthy, time.Second, 100*time.Millisecond)
}

func TestGetChannel_WhenHealthy(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: rabbitmqURI,
		}))
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, shell.WaitForHealth(ctx))

	ch, err := shell.GetChannel()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestGetChannel_WhenUnhealthy(t *testing.T) {
	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              "amqp://invalid:invalid@localhost:9999/",
			MinRetryInterval: seconds(1),
			MaxRetryInterval: seconds(2),
		}),
		rmqinitr.WithLazyConnect())
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ch, err := shell.GetChannel()
	assert.Error(t, err)
	assert.Nil(t, ch)
	assert.Equal(t, rmqinitr.ErrNotHealthy, err)
}

func TestGetChannel_AfterClose(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: rabbitmqURI,
		}))
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, shell.WaitForHealth(ctx))

	require.NoError(t, shell.Close(context.Background()))

	ch, err := shell.GetChannel()
	assert.Error(t, err)
	assert.Nil(t, ch)
	assert.Equal(t, rmqinitr.ErrShellClosed, err)
}

func TestReconnection_AfterConnectionLoss(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              rabbitmqURI,
			MaxRetryInterval: seconds(4),
			MinRetryInterval: seconds(1),
		}))
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	require.NoError(t, shell.WaitForHealth(ctx))

	// Stop and restart the broker application in place so the mapped port is
	// unchanged; the shell must observe the loss and reconnect.
	execInContainer(t, ctx, "rabbitmqctl", "stop_app")
	require.Eventually(t, func() bool { return !shell.IsHealthy() },
		15*time.Second, 100*time.Millisecond,
		"shell should observe the stopped broker")

	execInContainer(t, ctx, "rabbitmqctl", "start_app")
	require.NoError(t, shell.WaitForHealth(ctx))
	assert.True(t, shell.IsHealthy())
}

// execInContainer runs a command in the shared broker and fails the test with
// the command output when it exits non-zero.
func execInContainer(t *testing.T, ctx context.Context, args ...string) {
	t.Helper()
	code, out, err := container.Exec(ctx, args)
	require.NoError(t, err)
	output, _ := io.ReadAll(out)
	if closer, ok := out.(io.Closer); ok {
		_ = closer.Close()
	}
	require.Zero(t, code, "exec %v: %s", args, output)
}

func TestConcurrentAccess(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: rabbitmqURI,
		}))
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, shell.WaitForHealth(ctx))

	// Test concurrent access to GetChannel and IsHealthy.
	done := make(chan bool)
	errs := make(chan error, 100)

	for range 10 {
		go func() {
			defer func() { done <- true }()
			for range 10 {
				ch, err := shell.GetChannel()
				if err != nil {
					errs <- err
					return
				}
				if ch == nil {
					errs <- fmt.Errorf("got nil channel")
					return
				}
			}
		}()
	}

	for range 5 {
		go func() {
			defer func() { done <- true }()
			for range 20 {
				shell.IsHealthy()
			}
		}()
	}

	for range 15 {
		<-done
	}

	select {
	case err := <-errs:
		t.Fatalf("Concurrent access error: %v", err)
	default:
	}
}

func TestWaitForHealth_Timeout(t *testing.T) {
	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              "amqp://invalid:invalid@localhost:9999/",
			MaxRetryInterval: seconds(5),
			MinRetryInterval: seconds(1),
		}),
		rmqinitr.WithLazyConnect())
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestWaitForHealth_Success(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: rabbitmqURI,
		}))
	require.NoError(t, err)
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	assert.NoError(t, err)
}

func TestClose_MultipleCallsSafe(t *testing.T) {
	requireDocker(t)

	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI: rabbitmqURI,
		}))
	require.NoError(t, err)

	// Multiple close calls should be safe.
	err1 := shell.Close(context.Background())
	err2 := shell.Close(context.Background())
	err3 := shell.Close(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}
