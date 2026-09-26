package rmqinitr_test

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/rmqinitr"
	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDefaultsAndInvalidConfigOverrideThroughNew(t *testing.T) {
	zero, one, four, five := 0, 1, 4, 5
	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithURI("amqp://localhost"),
		rmqinitr.WithLazyConnect(),
	)
	if err != nil {
		t.Fatalf("New() with defaults: %v", err)
	}
	if shell.IsHealthy() {
		t.Fatal("lazy shell unexpectedly reported healthy before connecting")
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	shell, err = rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              "amqp://localhost",
			MinRetryInterval: &zero,
			MaxRetryInterval: &one,
		}),
		rmqinitr.WithMinRetryInterval(time.Second),
		rmqinitr.WithMaxRetryInterval(5*time.Second),
		rmqinitr.WithLazyConnect(),
	)
	if err != nil {
		t.Fatalf("New() did not apply later retry options before validation: %v", err)
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	shell, err = rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              "amqp://localhost",
			MinRetryInterval: &zero,
			MaxRetryInterval: &four,
		}),
		rmqinitr.WithConfig(&rmqinitr.Config{
			URI:              "amqp://localhost",
			MinRetryInterval: &one,
			MaxRetryInterval: &five,
		}),
		rmqinitr.WithLazyConnect(),
	)
	if err != nil {
		t.Fatalf("New() did not let later config values override earlier invalid values: %v", err)
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestOptionPrecedenceThroughNativeDial(t *testing.T) {
	dialed := make(chan string, 1)
	dialErr := errors.New("stop before network connection")
	shell, err := rmqinitr.New(context.Background(),
		rmqinitr.WithConfig(&rmqinitr.Config{URI: "amqp://config-host:5672"}),
		rmqinitr.WithURI("amqp://option-host:5673"),
		rmqinitr.WithLazyConnect(),
		rmqinitr.WithNativeDialConfig(func(config *amqp.Config) error {
			config.Heartbeat = 7 * time.Second
			config.Dial = func(_ string, address string) (net.Conn, error) {
				dialed <- address
				return nil, dialErr
			}
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer shell.Close(context.Background())

	select {
	case address := <-dialed:
		if address != "option-host:5673" {
			t.Errorf("native dial address = %q, want option-host:5673", address)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("native AMQP dial option was not used")
	}
}

func TestConfigValidationAndOptionOverrides(t *testing.T) {
	zero, one, four, five := 0, 1, 4, 5
	for _, tc := range []struct {
		name   string
		config *rmqinitr.Config
		want   string
	}{
		{name: "minimum must be positive", config: &rmqinitr.Config{URI: "amqp://localhost", MinRetryInterval: &zero}, want: "min retry interval"},
		{name: "maximum must exceed one second", config: &rmqinitr.Config{URI: "amqp://localhost", MaxRetryInterval: &one}, want: "max retry interval"},
		{name: "maximum must not be lower than minimum", config: &rmqinitr.Config{URI: "amqp://localhost", MinRetryInterval: &five, MaxRetryInterval: &four}, want: "lower than min"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := rmqinitr.New(context.Background(), rmqinitr.WithConfig(tc.config), rmqinitr.WithLazyConnect())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("New() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestNewRequiresURI(t *testing.T) {
	_, err := rmqinitr.New(context.Background(), rmqinitr.WithLazyConnect())
	if err == nil || !strings.Contains(err.Error(), "no rabbitmq configuration") {
		t.Fatalf("New() error = %v, want missing URI error", err)
	}
}
