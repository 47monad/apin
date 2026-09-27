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

func TestMustLoadPanicsOnError(t *testing.T) {
	var destination applicationConfig
	defer func() {
		if recover() == nil {
			t.Fatal("MustLoad() did not panic on load error")
		}
	}()
	config.MustLoad("missing.json", "", &destination)
}
