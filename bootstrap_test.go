package apin_test

import (
	"testing"

	"github.com/47monad/apin"
	"github.com/go-logr/logr"
)

func TestNew(t *testing.T) {
	app := apin.New()
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

func TestWithLoggerAndSetLogger(t *testing.T) {
	sink := &recordingSink{}
	app := apin.New(apin.WithLogger(logr.New(sink)))
	app.Logger().Info("hello")
	if sink.writes != 1 {
		t.Errorf("registered logger not in use, writes = %d", sink.writes)
	}

	// SetLogger swaps the logger after construction.
	app.SetLogger(logr.Discard())
}
