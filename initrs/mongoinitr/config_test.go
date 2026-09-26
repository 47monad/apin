package mongoinitr_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/47monad/apin/initrs/mongoinitr"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestConfigAndOptionPrecedenceThroughNew(t *testing.T) {
	stop := errors.New("stop before connect")
	var gotURI string

	_, err := mongoinitr.New(context.Background(),
		mongoinitr.WithConfig(&mongoinitr.Config{URI: "mongodb://config:27017", DBName: "config-db"}),
		mongoinitr.WithURI("mongodb://option:27017"),
		mongoinitr.WithDBName("option-db"),
		mongoinitr.WithNativeClientOptions(func(config *options.ClientOptions) error {
			config.SetDirect(true)
			return nil
		}),
		mongoinitr.WithNativeClientOptions(func(config *options.ClientOptions) error {
			gotURI = config.GetURI()
			if config.Direct == nil || !*config.Direct {
				t.Error("later native option did not observe earlier native option")
			}
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native option sentinel", err)
	}
	if gotURI != "mongodb://option:27017" {
		t.Errorf("URI = %q, want option URI", gotURI)
	}

}

func TestDefaultsAndNativeClientOptionsThroughNew(t *testing.T) {
	stop := errors.New("stop before connect")
	var connectTimeout *time.Duration
	var uri string

	_, err := mongoinitr.New(context.Background(),
		mongoinitr.WithNativeClientOptions(func(config *options.ClientOptions) error {
			connectTimeout = config.ConnectTimeout
			uri = config.GetURI()
			config.SetDirect(true)
			return stop
		}),
	)
	if !errors.Is(err, stop) {
		t.Fatalf("New() error = %v, want native option sentinel", err)
	}
	if connectTimeout != nil {
		t.Errorf("driver connect timeout = %v, want unset default", *connectTimeout)
	}
	if uri != "" {
		t.Errorf("URI = %q, want empty default URI", uri)
	}
}

func TestValidationThroughNew(t *testing.T) {
	tests := []struct {
		name string
		opts []mongoinitr.Option
		want string
	}{
		{name: "invalid URI", opts: []mongoinitr.Option{mongoinitr.WithConfig(&mongoinitr.Config{URI: "not a mongodb uri"})}, want: "MongoDB"},
		{name: "non-positive ping timeout", opts: []mongoinitr.Option{mongoinitr.WithPingTimeout(0)}, want: "ping timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := mongoinitr.New(context.Background(), tt.opts...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("New() error = %v, want substring %q", err, tt.want)
			}
		})
	}
}
