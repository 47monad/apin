package manifest_test

import (
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigStructure(t *testing.T) {
	t.Run("LoggingConfig_struct", func(t *testing.T) {
		cfg := manifest.LoggingConfig{
			Level: "debug",
		}
		assert.Equal(t, "debug", cfg.Level)
	})

	t.Run("HTTPConfig_struct", func(t *testing.T) {
		cfg := manifest.HTTPConfig{
			Servers: map[string]manifest.HTTPServerConfig{
				"main": {
					Port: 8080,
				},
			},
		}
		assert.Equal(t, 8080, cfg.Servers["main"].Port)
	})

	t.Run("full_struct", func(t *testing.T) {
		cfg := manifest.Config{
			Name:    "test-app",
			Title:   "Test Application",
			Version: "1.0.0",
			Env:     "development",
			Mode:    "debug",
			Host:    "localhost",
			Logging: manifest.LoggingConfig{
				Level: "debug",
			},
			HTTP: &manifest.HTTPConfig{
				Servers: map[string]manifest.HTTPServerConfig{
					"main": {
						Port: 8080,
					},
				},
			},
		}

		assert.Equal(t, "test-app", cfg.Name)
		assert.Equal(t, "Test Application", cfg.Title)
		assert.Equal(t, "1.0.0", cfg.Version)
		assert.Equal(t, "development", cfg.Env)
		assert.Equal(t, "debug", cfg.Mode)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, "debug", cfg.Logging.Level)

		require.NotNil(t, cfg.HTTP)
		assert.Equal(t, 8080, cfg.HTTP.Servers["main"].Port)
	})

	t.Run("nil_optional_fields", func(t *testing.T) {
		cfg := manifest.Config{
			Name:    "minimal-app",
			Title:   "Minimal Application",
			Version: "1.0.0",
			Env:     "production",
			Mode:    "release",
			Host:    "0.0.0.0",
			Logging: manifest.LoggingConfig{
				Level: "info",
			},
		}

		assert.Equal(t, "minimal-app", cfg.Name)
		assert.Equal(t, "Minimal Application", cfg.Title)
		assert.Nil(t, cfg.HTTP)
	})
}
