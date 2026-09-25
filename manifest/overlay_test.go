package manifest

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
)

var errOpenBoom = errors.New("open boom")

type boomFS struct{}

func (boomFS) Open(string) (fs.File, error) {
	return nil, errOpenBoom
}

func TestGetOverlayWalkErrorWraps(t *testing.T) {
	_, err := getOverlay(boomFS{})
	if err == nil {
		t.Fatal("getOverlay() succeeded on a failing filesystem")
	}
	if !strings.Contains(err.Error(), "walkdir") {
		t.Fatalf("getOverlay() error %q does not mention walkdir", err)
	}
	if !errors.Is(err, errOpenBoom) {
		t.Fatalf("getOverlay() error %v does not unwrap to the walk cause", err)
	}
}
