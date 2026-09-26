package prominitr_test

import (
	"testing"

	"github.com/47monad/apin/initrs/prominitr"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewConfigurationAndOptionPrecedence(t *testing.T) {
	defaultShell, err := prominitr.New(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if defaultShell.GRPCServerInterceptor != nil || defaultShell.GRPCServerMetrics != nil {
		t.Fatal("gRPC metrics are enabled by default")
	}
	if defaultShell.Registry == nil {
		t.Fatal("New returned a nil registry")
	}

	configuredShell, err := prominitr.New(t.Context(), prominitr.WithConfig(&prominitr.Config{GRPCMetrics: true}))
	if err != nil {
		t.Fatal(err)
	}
	if configuredShell.GRPCServerInterceptor == nil || configuredShell.GRPCServerMetrics == nil {
		t.Fatal("WithConfig did not enable gRPC instrumentation")
	}

	overriddenShell, err := prominitr.New(t.Context(),
		prominitr.WithConfig(&prominitr.Config{GRPCMetrics: true}),
		prominitr.WithGRPCMetrics(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if overriddenShell.GRPCServerInterceptor != nil || overriddenShell.GRPCServerMetrics != nil {
		t.Fatal("later WithGRPCMetrics(false) did not override Config")
	}

	enabledShell, err := prominitr.New(t.Context(), prominitr.WithGRPCMetrics(true))
	if err != nil {
		t.Fatal(err)
	}
	if enabledShell.GRPCServerInterceptor == nil || enabledShell.GRPCServerMetrics == nil {
		t.Fatal("WithGRPCMetrics(true) did not provide gRPC instrumentation")
	}
}

func TestWithPromMonitoring(t *testing.T) {
	interceptor, metrics := prominitr.WithPromMonitoring(prometheus.NewRegistry())
	if interceptor == nil {
		t.Fatal("WithPromMonitoring returned a nil interceptor")
	}
	if metrics == nil {
		t.Fatal("WithPromMonitoring returned nil metrics")
	}
}
