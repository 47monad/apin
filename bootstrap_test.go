package apin_test

import (
	"os"
	"testing"

	"github.com/47monad/apin"
	"github.com/go-logr/logr"
)

func TestNew(t *testing.T) {
	app, err := apin.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if app == nil {
		t.Fatal("New() app = nil")
	}
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
	app, err := apin.New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	sink := &recordingSink{}
	app.RegisterLogger(logr.New(sink))
	app.Logger().Info("hello")
	if sink.writes != 1 {
		t.Errorf("registered logger not in use, writes = %d", sink.writes)
	}

	// SetLogger swaps the logger again; RegisterLogger(nil) is a no-op.
	app.SetLogger(logr.Discard())
	app.RegisterLogger(logr.Logger{})
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
