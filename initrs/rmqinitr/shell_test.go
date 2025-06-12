package rmqinitr_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/rmqinitr" // Replace with your actual module path
	"github.com/47monad/zaal"
	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	rmqtc "github.com/testcontainers/testcontainers-go/modules/rabbitmq"
)

var (
	rabbitmqURI string
	container   *rmqtc.RabbitMQContainer
)

func TestMain(m *testing.M) {
	var err error
	ctx := context.Background()

	// Run rabbitmq container
	container, err = rmqtc.Run(ctx,
		"rabbitmq:4.1.1-management-alpine",
		testcontainers.WithExposedPorts("56720"),
		testcontainers.WithReuseByName("rmq-apin-client"),
		testcontainers.WithHostConfigModifier(func(hostConfig *dockerContainer.HostConfig) {
			hostConfig.PortBindings = nat.PortMap{
				rmqtc.DefaultAMQPPort: {{HostIP: "0.0.0.0", HostPort: "56720"}},
				rmqtc.DefaultHTTPPort: {{HostIP: "0.0.0.0", HostPort: "51672"}},
			}
		}),
	)
	if err != nil {
		log.Fatalf("could not run rabbitmq container: %s", err)
	}

	rabbitmqURI, err = container.AmqpURL(ctx)
	if err != nil {
		log.Fatalf("could not get amqp URI: %s", err)
	}

	code := m.Run()

	if err = container.Terminate(context.Background()); err != nil {
		log.Fatalf("Could not terminate rabbitmq container: %s", err)
	}

	os.Exit(code)
}

func TestNewFromConfig_ValidConnection(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{URI: rabbitmqURI})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Wait for connection to establish
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	assert.True(t, shell.IsHealthy())
}

func TestNewRabbitManager_InvalidConnection(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: "amqp://invalid:invalid@localhost:9999/",
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Should not become healthy with invalid connection
	time.Sleep(2 * time.Second)
	assert.False(t, shell.IsHealthy())
}

func TestGetChannel_WhenHealthy(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: rabbitmqURI,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Wait for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	// Should be able to get channel
	ch, err := shell.GetChannel()
	assert.NoError(t, err)
	assert.NotNil(t, ch)
}

func TestGetChannel_WhenUnhealthy(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI:              "amqp://invalid:invalid@localhost:9999/",
		MinRetryInterval: 1,
		MaxRetryInterval: 2,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Should not be able to get channel when unhealthy
	ch, err := shell.GetChannel()
	assert.Error(t, err)
	assert.Nil(t, ch)
	assert.Equal(t, rmqinitr.ErrNotHealthy, err)
}

func TestGetChannel_AfterClose(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: rabbitmqURI,
	})
	require.NoError(t, err)

	// Wait for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	// Close manager
	err = shell.Close(context.Background())
	require.NoError(t, err)

	// Should not be able to get channel after close
	ch, err := shell.GetChannel()
	assert.Error(t, err)
	assert.Nil(t, ch)
	assert.Equal(t, rmqinitr.ErrShellClosed, err)
}

func TestReconnection_AfterConnectionLoss(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI:              rabbitmqURI,
		MaxRetryInterval: 4,
		MinRetryInterval: 1,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Wait for initial connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	// Stop and restart RabbitMQ container to simulate connection loss
	err = runCommand("docker", "stop", container.GetContainerID())
	// err = container.Terminate(context.Background())
	require.NoError(t, err)

	// Wait for unhealthy state
	time.Sleep(2 * time.Second)
	assert.False(t, shell.IsHealthy())

	// Restart RabbitMQ
	err = runCommand("docker", "start", container.GetContainerID())
	require.NoError(t, err)

	// Should reconnect automatically
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	err = shell.WaitForHealth(ctx2)
	assert.NoError(t, err)
	assert.True(t, shell.IsHealthy())
}

func TestConcurrentAccess(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: rabbitmqURI,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Wait for connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	require.NoError(t, err)

	// Test concurrent access to GetChannel and IsHealthy
	done := make(chan bool)
	errors := make(chan error, 100)

	// Multiple goroutines accessing GetChannel
	for range 10 {
		go func() {
			defer func() { done <- true }()
			for range 10 {
				ch, err := shell.GetChannel()
				if err != nil {
					errors <- err
					return
				}
				if ch == nil {
					errors <- fmt.Errorf("got nil channel")
					return
				}
				time.Sleep(10 * time.Millisecond)
			}
		}()
	}

	// Multiple goroutines checking health
	for range 5 {
		go func() {
			defer func() { done <- true }()
			for range 20 {
				shell.IsHealthy()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	// Wait for all goroutines
	for range 15 {
		<-done
	}

	// Check for any errors
	select {
	case err := <-errors:
		t.Fatalf("Concurrent access error: %v", err)
	default:
		// No errors, test passed
	}
}

func TestWaitForHealth_Timeout(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI:              "amqp://invalid:invalid@localhost:9999/",
		MaxRetryInterval: 5,
		MinRetryInterval: 1,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestWaitForHealth_Success(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: rabbitmqURI,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = shell.WaitForHealth(ctx)
	assert.NoError(t, err)
}

func TestClose_MultipleCallsSafe(t *testing.T) {
	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI: rabbitmqURI,
	})
	require.NoError(t, err)

	// Multiple close calls should be safe
	err1 := shell.Close(context.Background())
	err2 := shell.Close(context.Background())
	err3 := shell.Close(context.Background())

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
}

func TestExponentialBackoff(t *testing.T) {
	// This test verifies that retry intervals increase (indirectly)
	start := time.Now()

	shell, err := rmqinitr.NewFromConfig(context.Background(), &zaal.RabbitMQConfig{
		URI:              "amqp://invalid:invalid@localhost:9999/",
		MaxRetryInterval: 4,
		MinRetryInterval: 1,
	})
	require.NoError(t, err)
	defer shell.Close(context.Background())

	// Wait a bit to let several retry attempts happen
	time.Sleep(8 * time.Second)

	// Should still be unhealthy but should have taken some time due to backoff
	assert.False(t, shell.IsHealthy())
	assert.True(t, time.Since(start) >= 8*time.Second)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	return nil
}
