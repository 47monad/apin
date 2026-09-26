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

	t.Run("RabbitMQConfig_struct", func(t *testing.T) {
		cfg := manifest.RabbitMQConfig{
			URI: "amqp://guest:guest@localhost:5672/",
		}
		assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.URI)
	})

	t.Run("PrometheusConfig_struct", func(t *testing.T) {
		cfg := manifest.PrometheusConfig{
			GRPCMetrics: true,
		}
		assert.True(t, cfg.GRPCMetrics)
	})

	t.Run("GRPCFeatures_struct", func(t *testing.T) {
		features := manifest.GRPCFeatures{
			Reflection:  true,
			HealthCheck: true,
			Logging:     true,
		}
		assert.True(t, features.Reflection)
		assert.True(t, features.HealthCheck)
		assert.True(t, features.Logging)
	})

	t.Run("GRPCClientConfig_struct", func(t *testing.T) {
		cfg := manifest.GRPCClientConfig{
			Address: "localhost:50051",
		}
		assert.Equal(t, "localhost:50051", cfg.Address)
	})

	t.Run("GRPCServerConfig_struct", func(t *testing.T) {
		cfg := manifest.GRPCServerConfig{
			Port: 50051,
			Features: manifest.GRPCFeatures{
				Reflection:  true,
				HealthCheck: true,
				Logging:     true,
			},
		}
		assert.Equal(t, 50051, cfg.Port)
		assert.True(t, cfg.Features.Reflection)
		assert.True(t, cfg.Features.HealthCheck)
		assert.True(t, cfg.Features.Logging)
	})

	t.Run("GRPCConfig_struct", func(t *testing.T) {
		cfg := manifest.GRPCConfig{
			Clients: map[string]manifest.GRPCClientConfig{
				"service1": {
					Address: "localhost:50051",
				},
			},
			Servers: map[string]manifest.GRPCServerConfig{
				"main": {
					Port: 50052,
					Features: manifest.GRPCFeatures{
						Reflection:  true,
						HealthCheck: true,
						Logging:     true,
					},
				},
			},
		}

		assert.Equal(t, "localhost:50051", cfg.Clients["service1"].Address)
		assert.Equal(t, 50052, cfg.Servers["main"].Port)
		assert.True(t, cfg.Servers["main"].Features.Reflection)
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
			RabbitMQ: &manifest.RabbitMQConfig{
				URI: "amqp://guest:guest@localhost:5672/",
			},
			Prometheus: &manifest.PrometheusConfig{
				GRPCMetrics: true,
			},
			GRPC: &manifest.GRPCConfig{
				Clients: map[string]manifest.GRPCClientConfig{
					"service1": {
						Address: "localhost:50051",
					},
				},
				Servers: map[string]manifest.GRPCServerConfig{
					"main": {
						Port: 50052,
						Features: manifest.GRPCFeatures{
							Reflection:  true,
							HealthCheck: true,
							Logging:     true,
						},
					},
				},
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

		require.NotNil(t, cfg.RabbitMQ)
		assert.Equal(t, "amqp://guest:guest@localhost:5672/", cfg.RabbitMQ.URI)

		require.NotNil(t, cfg.Prometheus)
		assert.True(t, cfg.Prometheus.GRPCMetrics)

		require.NotNil(t, cfg.GRPC)
		assert.Equal(t, "localhost:50051", cfg.GRPC.Clients["service1"].Address)
		assert.Equal(t, 50052, cfg.GRPC.Servers["main"].Port)

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
		assert.Nil(t, cfg.RabbitMQ)
		assert.Nil(t, cfg.Prometheus)
		assert.Nil(t, cfg.GRPC)
		assert.Nil(t, cfg.HTTP)
	})
}
