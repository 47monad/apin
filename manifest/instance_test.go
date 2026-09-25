package manifest

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/build"
	cueload "cuelang.org/go/cue/load"
)

func TestFirstInstanceEmpty(t *testing.T) {
	_, err := firstInstance(nil, "embedded schema")
	if err == nil {
		t.Fatal("firstInstance(nil) succeeded")
	}
	if !strings.Contains(err.Error(), "no CUE instances loaded for embedded schema") {
		t.Fatalf("firstInstance(nil) error %q", err)
	}

	_, err = firstInstance([]*build.Instance{}, "user config")
	if err == nil {
		t.Fatal("firstInstance(empty) succeeded")
	}
	if !strings.Contains(err.Error(), "no CUE instances loaded for user config") {
		t.Fatalf("firstInstance(empty) error %q", err)
	}

	_, err = firstInstance([]*build.Instance{nil}, "embedded schema")
	if err == nil {
		t.Fatal("firstInstance([nil]) succeeded")
	}
	if !strings.Contains(err.Error(), "nil CUE instance for embedded schema") {
		t.Fatalf("firstInstance([nil]) error %q", err)
	}
}

func TestFirstInstanceOK(t *testing.T) {
	ins := cueload.Instances([]string{"."}, &cueload.Config{Dir: "./cue/"})
	got, err := firstInstance(ins, "embedded schema")
	if err != nil {
		t.Fatalf("firstInstance() on a real load: %v", err)
	}
	if got == nil || got != ins[0] {
		t.Fatalf("firstInstance() = %v, want ins[0]", got)
	}
}
