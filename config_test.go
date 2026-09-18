package apin_test

import (
	"testing"

	"github.com/47monad/apin"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		envPath    string
		wantName   string
		wantErr    bool
	}{
		{
			name:       "success",
			configPath: "manifest/testdata/main.cue",
			envPath:    "manifest/testdata/main.env",
			wantName:   "test",
		},
		{
			name:       "bad_manifest_path",
			configPath: "nonexistent.cue",
			envPath:    "nonexistent.env",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := apin.LoadConfig(tt.configPath, tt.envPath)
			if tt.wantErr {
				if err == nil {
					t.Fatal("LoadConfig() error = nil, want error")
				}
				if cfg != nil {
					t.Fatalf("LoadConfig() cfg = %v, want nil", cfg)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if cfg == nil {
				t.Fatal("LoadConfig() cfg = nil")
			}
			if cfg.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", cfg.Name, tt.wantName)
			}
		})
	}
}

func TestMustLoadConfig(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cfg := apin.MustLoadConfig("manifest/testdata/main.cue", "manifest/testdata/main.env")
		if cfg == nil {
			t.Fatal("MustLoadConfig() cfg = nil")
		}
		if cfg.Name != "test" {
			t.Errorf("Name = %q, want %q", cfg.Name, "test")
		}
	})
	t.Run("bad_manifest_path_panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("MustLoadConfig() did not panic on bad manifest path")
			}
		}()
		_ = apin.MustLoadConfig("nonexistent.cue", "nonexistent.env")
	})
}
