package etcdinitr_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/etcdinitr"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestConfigAndOptionPrecedenceThroughNew(t *testing.T) {
	stop := errors.New("stop before client creation")
	var captured *clientv3.Config
	timeout := 3

	_, err := etcdinitr.New(context.Background(),
		etcdinitr.WithConfig(&etcdinitr.Config{
			Endpoints: "config-a:2379,config-b:2379",
			Username:  "config-user",
			Password:  "config-pass",
			Timeout:   &timeout,
		}),
		etcdinitr.WithEndpoints([]string{"option:2379"}),
		etcdinitr.WithUsername("option-user"),
		etcdinitr.WithTimeout(5*time.Second),
		etcdinitr.WithNativeConfig(func(config *clientv3.Config) error {
			copy := *config
			captured = &copy
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native option sentinel", err)
	}
	if !reflect.DeepEqual(captured.Endpoints, []string{"option:2379"}) {
		t.Errorf("Endpoints = %v, want [option:2379]", captured.Endpoints)
	}
	if captured.Username != "option-user" || captured.Password != "config-pass" {
		t.Errorf("credentials = %q/%q, want option-user/config-pass", captured.Username, captured.Password)
	}
	if captured.DialTimeout != 5*time.Second {
		t.Errorf("DialTimeout = %v, want 5s", captured.DialTimeout)
	}
}

func TestDefaultConfigAndNativeOptionThroughNew(t *testing.T) {
	stop := errors.New("stop before client creation")
	var captured *clientv3.Config

	_, err := etcdinitr.New(context.Background(), etcdinitr.WithNativeConfig(func(config *clientv3.Config) error {
		copy := *config
		captured = &copy
		config.DialTimeout = time.Second
		return stop
	}))
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native option sentinel", err)
	}
	if captured == nil {
		t.Fatal("native config option was not called")
	}
	if len(captured.Endpoints) != 0 || captured.DialTimeout != 0 {
		t.Errorf("default native config = %+v, want zero values", captured)
	}
}

func TestValidationOccursAfterAllOptions(t *testing.T) {
	stop := errors.New("stop before client creation")
	zero := 0

	_, err := etcdinitr.New(context.Background(),
		etcdinitr.WithConfig(&etcdinitr.Config{Timeout: &zero}),
		etcdinitr.WithTimeout(2*time.Second),
		etcdinitr.WithNativeConfig(func(*clientv3.Config) error { return stop }),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want later option sentinel after overriding invalid config timeout", err)
	}

	shell, err := etcdinitr.New(context.Background(),
		etcdinitr.WithConfig(&etcdinitr.Config{Timeout: &zero}),
		etcdinitr.WithEndpoints([]string{"127.0.0.1:2379"}),
		etcdinitr.WithNativeConfig(func(config *clientv3.Config) error {
			config.DialTimeout = 3 * time.Second
			return nil
		}),
	)
	if err != nil {
		t.Fatalf("New() error = %v, want native option to repair invalid config timeout", err)
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestInvalidTimeoutsThroughNew(t *testing.T) {
	zero := 0
	tests := []struct {
		name string
		opts []etcdinitr.Option
		want string
	}{
		{name: "config timeout", opts: []etcdinitr.Option{etcdinitr.WithConfig(&etcdinitr.Config{Timeout: &zero})}, want: "positive"},
		{name: "negative dial timeout", opts: []etcdinitr.Option{etcdinitr.WithEndpoints([]string{"127.0.0.1:2379"}), etcdinitr.WithTimeout(-time.Second)}, want: "must not be negative"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := etcdinitr.New(context.Background(), tt.opts...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestNewFromOptionsExposesNativeClient(t *testing.T) {
	shell, err := etcdinitr.New(context.Background(),
		etcdinitr.WithEndpoints([]string{"127.0.0.1:2379"}),
		etcdinitr.WithTimeout(time.Second),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if shell.Client == nil {
		t.Fatal("New() returned nil native client")
	}
	if err := shell.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
