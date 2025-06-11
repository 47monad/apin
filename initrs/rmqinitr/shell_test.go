package rmqinitr_test

// import (
// 	"context"
// 	"fmt"
// 	"log"
// 	"os"
// 	"testing"
// 	"time"
//
// 	"github.com/ory/dockertest/v3"
// 	"github.com/ory/dockertest/v3/docker"
// 	amqp "github.com/rabbitmq/amqp091-go"
// 	"github.com/stretchr/testify/assert"
// 	"github.com/stretchr/testify/require"
//
// 	"github.com/47monad/apin/initrs/rmqinitr" // Replace with your actual module path
// )
//
// var (
// 	rabbitmqURL string
// 	pool        *dockertest.Pool
// 	resource    *dockertest.Resource
// )
//
// func TestMain(m *testing.M) {
// 	var err error
// 	pool, err = dockertest.NewPool("")
// 	if err != nil {
// 		log.Fatalf("Could not connect to docker: %s", err)
// 	}
//
// 	// Start RabbitMQ container
// 	resource, err = pool.RunWithOptions(&dockertest.RunOptions{
// 		Repository: "rabbitmq",
// 		Tag:        "3.12-management-alpine",
// 		Env: []string{
// 			"RABBITMQ_DEFAULT_USER=test",
// 			"RABBITMQ_DEFAULT_PASS=test",
// 		},
// 	}, func(config *docker.HostConfig) {
// 		config.AutoRemove = true
// 		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
// 	})
// 	if err != nil {
// 		log.Fatalf("Could not start resource: %s", err)
// 	}
//
// 	hostAndPort := resource.GetHostPort("5672/tcp")
// 	rabbitmqURL = fmt.Sprintf("amqp://test:test@%s/", hostAndPort)
//
// 	// Wait for RabbitMQ to be ready
// 	pool.MaxWait = 120 * time.Second
// 	if err = pool.Retry(func() error {
// 		conn, err := amqp.Dial(rabbitmqURL)
// 		if err != nil {
// 			return err
// 		}
// 		return conn.Close()
// 	}); err != nil {
// 		log.Fatalf("Could not connect to RabbitMQ: %s", err)
// 	}
//
// 	code := m.Run()
//
// 	if err := pool.Purge(resource); err != nil {
// 		log.Fatalf("Could not purge resource: %s", err)
// 	}
//
// 	os.Exit(code)
// }
//
// func TestNewRabbitManager_ValidConnection(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
// 	defer mgr.Close()
//
// 	// Wait for connection to establish
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	require.NoError(t, err)
//
// 	assert.True(t, mgr.IsHealthy())
// }
//
// func TestNewRabbitManager_InvalidConnection(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager("amqp://invalid:invalid@localhost:9999/", &rmqinitr.Config{
// 		MaxRetryInterval: 1 * time.Second,
// 		MinRetryInterval: 100 * time.Millisecond,
// 	})
// 	defer mgr.Close()
//
// 	// Should not become healthy with invalid connection
// 	time.Sleep(2 * time.Second)
// 	assert.False(t, mgr.IsHealthy())
// }
//
// func TestGetChannel_WhenHealthy(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
// 	defer mgr.Close()
//
// 	// Wait for connection
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	require.NoError(t, err)
//
// 	// Should be able to get channel
// 	ch, err := mgr.GetChannel()
// 	assert.NoError(t, err)
// 	assert.NotNil(t, ch)
// }
//
// func TestGetChannel_WhenUnhealthy(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager("amqp://invalid:invalid@localhost:9999/", &rmqinitr.Config{
// 		MaxRetryInterval: 1 * time.Second,
// 		MinRetryInterval: 100 * time.Millisecond,
// 	})
// 	defer mgr.Close()
//
// 	// Should not be able to get channel when unhealthy
// 	ch, err := mgr.GetChannel()
// 	assert.Error(t, err)
// 	assert.Nil(t, ch)
// 	assert.Equal(t, rmqinitr.ErrNotHealthy, err)
// }
//
// func TestGetChannel_AfterClose(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
//
// 	// Wait for connection
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	require.NoError(t, err)
//
// 	// Close manager
// 	err = mgr.Close()
// 	require.NoError(t, err)
//
// 	// Should not be able to get channel after close
// 	ch, err := mgr.GetChannel()
// 	assert.Error(t, err)
// 	assert.Nil(t, ch)
// 	assert.Equal(t, rmqinitr.ErrManagerClosed, err)
// }
//
// func TestReconnection_AfterConnectionLoss(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, &rmqinitr.Config{
// 		MaxRetryInterval: 2 * time.Second,
// 		MinRetryInterval: 100 * time.Millisecond,
// 	})
// 	defer mgr.Close()
//
// 	// Wait for initial connection
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	require.NoError(t, err)
//
// 	// Stop and restart RabbitMQ container to simulate connection loss
// 	err = pool.Purge(resource)
// 	require.NoError(t, err)
//
// 	// Wait for unhealthy state
// 	time.Sleep(1 * time.Second)
// 	assert.False(t, mgr.IsHealthy())
//
// 	// Restart RabbitMQ
// 	resource, err = pool.RunWithOptions(&dockertest.RunOptions{
// 		Repository: "rabbitmq",
// 		Tag:        "3.12-management-alpine",
// 		Env: []string{
// 			"RABBITMQ_DEFAULT_USER=test",
// 			"RABBITMQ_DEFAULT_PASS=test",
// 		},
// 	}, func(config *docker.HostConfig) {
// 		config.AutoRemove = true
// 		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
// 	})
// 	require.NoError(t, err)
//
// 	// Wait for RabbitMQ to be ready again
// 	pool.MaxWait = 60 * time.Second
// 	err = pool.Retry(func() error {
// 		conn, err := amqp.Dial(rabbitmqURL)
// 		if err != nil {
// 			return err
// 		}
// 		return conn.Close()
// 	})
// 	require.NoError(t, err)
//
// 	// Should reconnect automatically
// 	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancel2()
//
// 	err = mgr.WaitForHealth(ctx2)
// 	assert.NoError(t, err)
// 	assert.True(t, mgr.IsHealthy())
// }
//
// func TestConcurrentAccess(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
// 	defer mgr.Close()
//
// 	// Wait for connection
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	require.NoError(t, err)
//
// 	// Test concurrent access to GetChannel and IsHealthy
// 	done := make(chan bool)
// 	errors := make(chan error, 100)
//
// 	// Multiple goroutines accessing GetChannel
// 	for i := 0; i < 10; i++ {
// 		go func() {
// 			defer func() { done <- true }()
// 			for j := 0; j < 10; j++ {
// 				ch, err := mgr.GetChannel()
// 				if err != nil {
// 					errors <- err
// 					return
// 				}
// 				if ch == nil {
// 					errors <- fmt.Errorf("got nil channel")
// 					return
// 				}
// 				time.Sleep(10 * time.Millisecond)
// 			}
// 		}()
// 	}
//
// 	// Multiple goroutines checking health
// 	for i := 0; i < 5; i++ {
// 		go func() {
// 			defer func() { done <- true }()
// 			for j := 0; j < 20; j++ {
// 				mgr.IsHealthy()
// 				time.Sleep(5 * time.Millisecond)
// 			}
// 		}()
// 	}
//
// 	// Wait for all goroutines
// 	for i := 0; i < 15; i++ {
// 		<-done
// 	}
//
// 	// Check for any errors
// 	select {
// 	case err := <-errors:
// 		t.Fatalf("Concurrent access error: %v", err)
// 	default:
// 		// No errors, test passed
// 	}
// }
//
// func TestWaitForHealth_Timeout(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager("amqp://invalid:invalid@localhost:9999/", &rmqinitr.Config{
// 		MaxRetryInterval: 5 * time.Second,
// 		MinRetryInterval: 1 * time.Second,
// 	})
// 	defer mgr.Close()
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	assert.Error(t, err)
// 	assert.Equal(t, context.DeadlineExceeded, err)
// }
//
// func TestWaitForHealth_Success(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
// 	defer mgr.Close()
//
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
//
// 	err := mgr.WaitForHealth(ctx)
// 	assert.NoError(t, err)
// }
//
// func TestClose_MultipleCallsSafe(t *testing.T) {
// 	mgr := rmqinitr.NewRabbitManager(rabbitmqURL, nil)
//
// 	// Multiple close calls should be safe
// 	err1 := mgr.Close()
// 	err2 := mgr.Close()
// 	err3 := mgr.Close()
//
// 	assert.NoError(t, err1)
// 	assert.NoError(t, err2)
// 	assert.NoError(t, err3)
// }
//
// func TestExponentialBackoff(t *testing.T) {
// 	// This test verifies that retry intervals increase (indirectly)
// 	start := time.Now()
//
// 	mgr := rmqinitr.NewRabbitManager("amqp://invalid:invalid@localhost:9999/", &rmqinitr.Config{
// 		MaxRetryInterval: 4 * time.Second,
// 		MinRetryInterval: 100 * time.Millisecond,
// 	})
// 	defer mgr.Close()
//
// 	// Wait a bit to let several retry attempts happen
// 	time.Sleep(8 * time.Second)
//
// 	// Should still be unhealthy but should have taken some time due to backoff
// 	assert.False(t, mgr.IsHealthy())
// 	assert.True(t, time.Since(start) >= 8*time.Second)
// }
