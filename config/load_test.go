package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/47monad/apin/config"
)

type applicationConfig struct {
	Name    string                  `json:"name" yaml:"name" env:"app_name"`
	Nested  nestedConfig            `json:"nested" yaml:"nested"`
	Servers map[string]serverConfig `json:"servers" yaml:"servers"`
}

type nestedConfig struct {
	Host string `json:"host" yaml:"host" env:"db_host"`
}

type serverConfig struct {
	Port int `json:"port" yaml:"port" env:"port"`
}

type pointerConfig struct {
	Nested *nestedConfig `json:"nested" yaml:"nested"`
}

type pointerIntConfig struct {
	Timeout *int `json:"timeout" yaml:"timeout" env:"test_timeout"`
}

// recursiveConfig reaches itself through a pointer, which the overlay must
// handle without materializing an unbounded chain of instances.
type recursiveConfig struct {
	Name string            `json:"name" yaml:"name" env:"rec_name"`
	Next *recursiveConfig  `json:"next" yaml:"next" env:"rec_next"`
	Kids []recursiveConfig `json:"kids" yaml:"kids" env:"rec_kids"`
}

func writeFile(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadJSONYAMLAndEnvironmentPrecedence(t *testing.T) {
	t.Setenv("APP_NAME", "from-process")
	t.Setenv("API_PORT", "9000")
	t.Setenv("DB_HOST", "")

	for _, tc := range []struct {
		name string
		ext  string
		data string
	}{
		{name: "json", ext: ".json", data: `{"name":"from-file","nested":{"host":"file-host"},"servers":{"api":{"port":80}}}`},
		{name: "yaml", ext: ".yaml", data: "name: from-file\nnested:\n  host: file-host\nservers:\n  api:\n    port: 80\n"},
		{name: "yml", ext: ".yml", data: "name: from-file\nnested:\n  host: file-host\nservers:\n  api:\n    port: 80\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configPath := writeFile(t, "app"+tc.ext, tc.data)
			envPath := writeFile(t, "app.env", "APP_NAME=from-file-env\nDB_HOST=env-host\nAPI_PORT=8080\n")
			var got applicationConfig
			if err := config.Load(configPath, envPath, &got); err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if got.Name != "from-process" || got.Nested.Host != "env-host" || got.Servers["api"].Port != 9000 {
				t.Errorf("loaded config = %+v, want process/file overlays", got)
			}
			if value, ok := os.LookupEnv("DB_HOST"); !ok || value != "" {
				t.Errorf("Load() changed process DB_HOST = %q, present = %v", value, ok)
			}
		})
	}
}

func TestLoadIgnoresMissingOptionalEnvFile(t *testing.T) {
	t.Setenv("APP_NAME", "from-process")
	configPath := writeFile(t, "app.json", `{"name":"from-file"}`)
	envPath := filepath.Join(t.TempDir(), ".env")
	var got applicationConfig
	if err := config.Load(configPath, envPath, &got); err != nil {
		t.Fatalf("Load() with missing optional env file error = %v", err)
	}
	if got.Name != "from-process" {
		t.Errorf("Name = %q, want process environment value", got.Name)
	}
}

func TestLoadStillReportsEnvPathErrorsOtherThanMissing(t *testing.T) {
	configPath := writeFile(t, "app.json", `{}`)
	envPath := t.TempDir()
	var got applicationConfig
	err := config.Load(configPath, envPath, &got)
	if err == nil || !strings.Contains(err.Error(), envPath) {
		t.Fatalf("Load() error = %v, want error containing env path %q", err, envPath)
	}
}

func TestLoadRejectsInvalidDocumentsAndDestination(t *testing.T) {
	tests := []struct {
		name string
		ext  string
		data string
		want string
	}{
		{name: "unknown JSON field", ext: ".json", data: `{"unknown":true}`, want: "unknown"},
		{name: "trailing JSON value", ext: ".json", data: `{} {}`, want: "trailing"},
		{name: "unknown YAML field", ext: ".yaml", data: "unknown: true\n", want: "unknown"},
		{name: "multiple YAML documents", ext: ".yml", data: "name: one\n---\nname: two\n", want: "multiple"},
		{name: "unsupported extension", ext: ".toml", data: "name = 'x'", want: ".toml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeFile(t, "app"+tc.ext, tc.data)
			var destination applicationConfig
			err := config.Load(path, "", &destination)
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), path) {
				t.Fatalf("Load() error = %v, want %q and file path %q", err, tc.want, path)
			}
		})
	}

	path := writeFile(t, "app.json", `{}`)
	if err := config.Load(path, "", applicationConfig{}); err == nil || !strings.Contains(err.Error(), "pointer") {
		t.Errorf("Load() with non-pointer destination error = %v, want pointer error", err)
	}
	var nilDestination *applicationConfig
	if err := config.Load(path, "", nilDestination); err == nil || !strings.Contains(err.Error(), "nil") {
		t.Errorf("Load() with nil destination error = %v, want nil error", err)
	}
}

func TestLoadEnvironmentConversionErrorHasFieldAndFileContext(t *testing.T) {
	configPath := writeFile(t, "app.json", `{"servers":{"api":{"port":80}}}`)
	envPath := writeFile(t, "app.env", "API_PORT=not-an-int\n")
	var destination applicationConfig
	err := config.Load(configPath, envPath, &destination)
	if err == nil || !strings.Contains(err.Error(), configPath) || !strings.Contains(err.Error(), "API_PORT") || !strings.Contains(err.Error(), "Servers[api].Port") {
		t.Fatalf("Load() error = %v, want file, env key, and field context", err)
	}
}

func TestLoadAppliesEnvironmentToOptionalPointer(t *testing.T) {
	t.Setenv("DB_HOST", "")
	configPath := writeFile(t, "app.json", `{}`)
	var withoutEnv pointerConfig
	if err := config.Load(configPath, "", &withoutEnv); err != nil {
		t.Fatalf("Load() without env error = %v", err)
	}
	if withoutEnv.Nested != nil {
		t.Fatalf("Load() created optional nested config without an environment value: %+v", withoutEnv.Nested)
	}

	envPath := writeFile(t, "app.env", "DB_HOST=env-host\n")
	var withEnv pointerConfig
	if err := config.Load(configPath, envPath, &withEnv); err != nil {
		t.Fatalf("Load() with env error = %v", err)
	}
	if withEnv.Nested == nil || withEnv.Nested.Host != "env-host" {
		t.Fatalf("Load() nested config = %+v, want host env-host", withEnv.Nested)
	}
}

func TestLoadAppliesEnvironmentToOptionalScalarPointer(t *testing.T) {
	t.Setenv("TEST_TIMEOUT", "9")
	configPath := writeFile(t, "pointer.json", `{"timeout":5}`)
	envPath := writeFile(t, "pointer.env", "TEST_TIMEOUT=8\n")
	var destination pointerIntConfig
	if err := config.Load(configPath, envPath, &destination); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if destination.Timeout == nil || *destination.Timeout != 9 {
		t.Fatalf("Load() timeout = %v, want process value 9", destination.Timeout)
	}

	t.Setenv("TEST_TIMEOUT", "")
	if err := config.Load(configPath, envPath, &destination); err != nil {
		t.Fatalf("Load() with empty process value error = %v", err)
	}
	if destination.Timeout == nil || *destination.Timeout != 8 {
		t.Fatalf("Load() timeout = %v, want dotenv value 8", destination.Timeout)
	}
}

// A recursive destination type must not send the overlay into unbounded
// recursion. Without a guard this overflows the stack in containsEnvType even
// with no environment at all, so the case has to terminate rather than crash.
func TestLoadRecursiveTypeWithoutEnvironmentTerminates(t *testing.T) {
	configPath := writeFile(t, "app.json", `{}`)
	var got recursiveConfig
	if err := config.Load(configPath, "", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Next != nil {
		t.Errorf("Load() materialized Next = %+v, want nil without an environment value", got.Next)
	}
	if len(got.Kids) != 0 {
		t.Errorf("Load() populated Kids = %+v, want empty", got.Kids)
	}
}

// A matching environment value justifies exactly one materialized instance,
// not an endless chain of them.
func TestLoadRecursiveTypeMaterializesOnlyOneInstance(t *testing.T) {
	t.Setenv("REC_NAME", "from-env")
	configPath := writeFile(t, "app.json", `{}`)
	var got recursiveConfig
	if err := config.Load(configPath, "", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Name != "from-env" {
		t.Errorf("Load() name = %q, want from-env", got.Name)
	}
	if got.Next == nil || got.Next.Name != "from-env" {
		t.Fatalf("Load() next = %+v, want one instance with name from-env", got.Next)
	}
	if got.Next.Next != nil {
		t.Errorf("Load() chained a second instance: %+v", got.Next.Next)
	}
}

// Recursive values that the file already supplied stay finite and still receive
// their overlays; only materializing new ones is bounded.
func TestLoadRecursiveTypeOverlaysExistingChain(t *testing.T) {
	t.Setenv("REC_NAME", "overlaid")
	configPath := writeFile(t, "app.json", `{"next":{"next":{"name":"deep"}}}`)

	var got recursiveConfig
	if err := config.Load(configPath, "", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	depth := 0
	for node := &got; node != nil; node = node.Next {
		depth++
		if depth > 8 {
			t.Fatalf("Load() produced a chain longer than the file supplied: %+v", got)
		}
		if node.Name != "overlaid" {
			t.Errorf("node %d name = %q, want overlaid", depth, node.Name)
		}
	}
	// The file supplies three nodes; the matching environment value justifies
	// one more at the tail, and the chain stops there.
	if depth != 4 {
		t.Errorf("chain length = %d, want 3 file nodes plus 1 materialized", depth)
	}
}

// Each map key puts the same recursive type under a different environment
// prefix, so both entries are materialized independently. This is what keeps
// the guard from simply blocking a repeated type.
func TestLoadRecursiveTypeUnderDistinctMapKeys(t *testing.T) {
	t.Setenv("PRIMARY_REC_NAME", "from-primary")
	t.Setenv("SECONDARY_REC_NAME", "from-secondary")
	configPath := writeFile(t, "app.json", `{"peers":{"primary":{},"secondary":{}}}`)

	var got struct {
		Peers map[string]recursiveConfig `json:"peers" yaml:"peers"`
	}
	if err := config.Load(configPath, "", &got); err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.Peers["primary"].Name != "from-primary" {
		t.Errorf("peers[primary].name = %q, want from-primary", got.Peers["primary"].Name)
	}
	if got.Peers["secondary"].Name != "from-secondary" {
		t.Errorf("peers[secondary].name = %q, want from-secondary", got.Peers["secondary"].Name)
	}
	if len(got.Peers) != 2 {
		t.Errorf("peers = %+v, want both entries", got.Peers)
	}
}

func TestMustLoadPanicsOnError(t *testing.T) {
	var destination applicationConfig
	defer func() {
		if recover() == nil {
			t.Fatal("MustLoad() did not panic on load error")
		}
	}()
	config.MustLoad("missing.json", "", &destination)
}
