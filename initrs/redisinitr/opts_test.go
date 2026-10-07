package redisinitr_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/redisinitr"
	"github.com/redis/go-redis/v9"
)

func TestConfigAndOptionPrecedenceThroughNew(t *testing.T) {
	stop := errors.New("stop before client creation")
	var captured *redis.UniversalOptions
	_, err := redisinitr.New(context.Background(),
		redisinitr.WithConfig(&redisinitr.Config{
			Addresses:  []string{"config-a:6379", "config-b:6379"},
			Username:   "config-user",
			Password:   "config-pass",
			Database:   3,
			MasterName: "config-master",
		}),
		redisinitr.WithAddresses([]string{"option:6379"}),
		redisinitr.WithUsername("option-user"),
		redisinitr.WithNativeOptions(func(opts *redis.UniversalOptions) error {
			copy := *opts
			captured = &copy
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native option sentinel", err)
	}
	if !reflect.DeepEqual(captured.Addrs, []string{"option:6379"}) {
		t.Errorf("Addrs = %v, want [option:6379]", captured.Addrs)
	}
	if captured.Username != "option-user" || captured.Password != "config-pass" {
		t.Errorf("credentials = %q/%q, want option-user/config-pass", captured.Username, captured.Password)
	}
	if captured.DB != 3 || captured.MasterName != "config-master" {
		t.Errorf("DB/MasterName = %d/%q, want 3/config-master", captured.DB, captured.MasterName)
	}
}

func TestValidation(t *testing.T) {
	for _, tt := range []struct {
		name string
		opts []redisinitr.Option
		want string
	}{
		{name: "no addresses", opts: nil, want: "at least one address"},
		{name: "blank address", opts: []redisinitr.Option{redisinitr.WithAddresses([]string{"127.0.0.1:6379", "  "})}, want: "must not be blank"},
		{name: "negative database", opts: []redisinitr.Option{redisinitr.WithAddresses([]string{"127.0.0.1:6379"}), redisinitr.WithDatabase(-1)}, want: "must not be negative"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, err := redisinitr.New(context.Background(), tt.opts...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestWithAddressesCopiesCallerSlice(t *testing.T) {
	addresses := []string{"127.0.0.1:6379"}
	var captured *redis.UniversalOptions
	shell, err := redisinitr.New(context.Background(),
		redisinitr.WithAddresses(addresses),
		redisinitr.WithNativeOptions(func(opts *redis.UniversalOptions) error {
			copy := *opts
			captured = &copy
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = shell.Close(context.Background()) }()

	addresses[0] = "mutated:6379"
	if !reflect.DeepEqual(captured.Addrs, []string{"127.0.0.1:6379"}) {
		t.Fatalf("Addrs = %v, want the copy made at construction time", captured.Addrs)
	}
}

func TestReadyReportsUnreachableServer(t *testing.T) {
	shell, err := redisinitr.New(context.Background(), redisinitr.WithAddresses([]string{"127.0.0.1:1"}))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer func() { _ = shell.Close(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	if err := shell.Ready(ctx); err == nil {
		t.Fatal("Ready() error = nil for an unreachable server, want error")
	}
}

func TestReadyNotInitialized(t *testing.T) {
	shell := &redisinitr.Shell{}
	if err := shell.Ready(context.Background()); err == nil {
		t.Fatal("Ready() error = nil on uninitialized shell, want error")
	}
}
