package prominitr_test

import (
	"strings"
	"testing"

	"github.com/47monad/apin/initrs/prominitr"
	"github.com/prometheus/client_golang/prometheus"
)

func hasMetricPrefix(t *testing.T, registry *prometheus.Registry, prefix string) bool {
	t.Helper()
	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if strings.HasPrefix(family.GetName(), prefix) {
			return true
		}
	}
	return false
}

func TestCollectorsAreOptIn(t *testing.T) {
	shell, err := prominitr.New(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if shell.Registry == nil {
		t.Fatal("New returned a nil registry")
	}
	if hasMetricPrefix(t, shell.Registry, "go_") {
		t.Fatal("Go collector is enabled by default")
	}
	if hasMetricPrefix(t, shell.Registry, "process_") {
		t.Fatal("process collector is enabled by default")
	}
}

func TestConfigAndOptionPrecedence(t *testing.T) {
	configured, err := prominitr.New(t.Context(), prominitr.WithConfig(&prominitr.Config{GoCollector: true}))
	if err != nil {
		t.Fatal(err)
	}
	if !hasMetricPrefix(t, configured.Registry, "go_") {
		t.Fatal("WithConfig did not enable the Go collector")
	}

	overridden, err := prominitr.New(t.Context(),
		prominitr.WithConfig(&prominitr.Config{GoCollector: true}),
		prominitr.WithGoCollector(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if hasMetricPrefix(t, overridden.Registry, "go_") {
		t.Fatal("later WithGoCollector(false) did not override Config")
	}

	process, err := prominitr.New(t.Context(), prominitr.WithProcessCollector(true))
	if err != nil {
		t.Fatal(err)
	}
	if !hasMetricPrefix(t, process.Registry, "process_") {
		t.Fatal("WithProcessCollector(true) did not register process metrics")
	}
}
