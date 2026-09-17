package apin_test

import (
	"os"
	"testing"

	"github.com/47monad/apin"
	"github.com/47monad/apin/manifest"
	"github.com/go-logr/logr"
)

func TestNewWithoutConfig(t *testing.T) {
	app, err := apin.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app.Config() != nil {
		t.Errorf("Config() = %v, want nil without WithConfig", app.Config())
	}
}

func TestNewLoadsManifest(t *testing.T) {
	app, err := apin.New(
		apin.WithConfig("manifest/testdata/main.cue"),
		apin.WithEnv("manifest/testdata/main.env"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	cfg := app.Config()
	if cfg == nil {
		t.Fatal("Config() = nil, want loaded manifest")
	}
	if cfg.Name != "test" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test")
	}
	// Env overlay must apply on top of the CUE values.
	if got, want := cfg.Logging.Level, "info"; got != want {
		t.Errorf("Logging.Level = %q, want %q from env", got, want)
	}
	if got, want := cfg.HTTP.Servers["main"].Port, 8787; got != want {
		t.Errorf("HTTP port = %d, want %d", got, want)
	}
}

func TestNewWithEnvAfterConfig(t *testing.T) {
	// WithEnv after WithConfig must still take effect: the manifest is
	// loaded only after all options are applied.
	app, err := apin.New(
		apin.WithEnv("manifest/testdata/main.env"),
		apin.WithConfig("manifest/testdata/main.cue"),
	)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if got, want := app.Config().Logging.Level, "info"; got != want {
		t.Errorf("Logging.Level = %q, want %q from env", got, want)
	}
}

func TestNewBadManifestPath(t *testing.T) {
	_, err := apin.New(apin.WithConfig("nonexistent.cue"))
	if err == nil {
		t.Fatal("New() error = nil, want manifest load error")
	}
}

func TestNewMissingEnvFileIsNotAnError(t *testing.T) {
	app, err := apin.New(
		apin.WithConfig("manifest/testdata/main.cue"),
		apin.WithEnv("nonexistent.env"),
	)
	if err != nil {
		t.Fatalf("New() error = %v, want nil for missing env file", err)
	}
	if app.Config() == nil {
		t.Fatal("Config() = nil, want loaded manifest")
	}
}

func TestMustNewPanicsOnBadManifest(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("MustNew() did not panic on bad manifest path")
		}
	}()
	apin.MustNew(apin.WithConfig("nonexistent.cue"))
}

// recordingSink captures log writes so tests can verify which logger the
// app holds.
type recordingSink struct {
	writes int
}

func (s *recordingSink) Enabled(int) bool { return true }

func (s *recordingSink) Init(logr.RuntimeInfo) {}

func (s *recordingSink) WithName(string) logr.LogSink { return s }

func (s *recordingSink) WithCallDepth(int) logr.LogSink { return s }

func (s *recordingSink) WithValues(...any) logr.LogSink { return s }

func (s *recordingSink) Info(int, string, ...any) { s.writes++ }

func (s *recordingSink) Error(error, string, ...any) { s.writes++ }

func TestRegisterLoggerAndSetLogger(t *testing.T) {
	app, err := apin.New(apin.WithConfig("manifest/testdata/main.cue"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sink := &recordingSink{}
	app.RegisterLogger(&apin.LoggerShell{Logger: logr.New(sink)})
	app.Logger().Info("hello")
	if sink.writes != 1 {
		t.Errorf("registered logger not in use, writes = %d", sink.writes)
	}

	// SetLogger swaps the logger again; RegisterLogger(nil) is a no-op.
	app.SetLogger(logr.Discard())
	app.RegisterLogger(nil)
}

func TestNewDecodesInitrSections(t *testing.T) {
	// The loaded manifest must decode into the same section types initrs
	// consume via WithConfig.
	app, err := apin.New(apin.WithConfig("manifest/testdata/main.cue"))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	cfg := app.Config()
	var (
		_ *manifest.GRPCConfig    = cfg.GRPC
		_ *manifest.HTTPConfig    = cfg.HTTP
		_ *manifest.EtcdConfig    = cfg.Etcd
		_ *manifest.MongodbConfig = cfg.Mongodb
	)
	if len(cfg.GRPC.Servers) == 0 {
		t.Error("GRPC.Servers empty, want decoded section")
	}
}

func TestLoadConfigStillWorks(t *testing.T) {
	// The standalone loader keeps working alongside the app-integrated one.
	cfg, err := apin.LoadConfig(
		"manifest/testdata/main.cue",
		"manifest/testdata/main.env",
	)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.Name != "test" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
