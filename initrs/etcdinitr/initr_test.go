package etcdinitr_test

import (
	"context"
	"testing"

	"github.com/47monad/apin/initrs/etcdinitr"
)

func TestShellReadyNotInitialized(t *testing.T) {
	shell := &etcdinitr.Shell{}
	if err := shell.Ready(context.Background()); err == nil {
		t.Fatal("Ready() error = nil on uninitialized shell, want error")
	}
}
