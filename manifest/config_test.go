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

	t.Run("MongodbOptions_struct", func(t *testing.T) {
		options := manifest.MongodbOptions{
			ReplicaSet: "rs0",
		}
		assert.Equal(t, "rs0", options.ReplicaSet)
	})

	t.Run("MongodbConfig_struct", func(t *testing.T) {
		cfg := manifest.MongodbConfig{
			URI:      "mongodb://localhost:27017",
			Username: "user",
			Password: "pass",
			DBName:   "testdb",
			Hosts:    []string{"localhost:27017", "localhost:27018"},
			Options: manifest.MongodbOptions{
				ReplicaSet: "rs0",
			},
		}

		assert.Equal(t, "mongodb://localhost:27017", cfg.URI)
		assert.Equal(t, "user", cfg.Username)
		assert.Equal(t, "pass", cfg.Password)
		assert.Equal(t, "testdb", cfg.DBName)
		assert.Equal(t, []string{"localhost:27017", "localhost:27018"}, cfg.Hosts)
		assert.Equal(t, "rs0", cfg.Options.ReplicaSet)
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

	t.Run("PostgresConfig_struct", func(t *testing.T) {
		cfg := manifest.PostgresConfig{
			URI:         "postgres://localhost:2231",
			Host:        "localhost",
			Port:        2231,
			Username:    "postgres",
			Password:    "secret",
			DBName:      "testdb",
			SSLMode:     "require",
			AppName:     "test-app",
			ConnTimeout: 5,
			Mode:        manifest.PostgresModePool,
			Pool: manifest.PostgresPoolConfig{
				MaxConns:            10,
				MinConns:            2,
				MaxConnLifetime:     300,
				MaxConnIdleTime:     60,
				HealthCheckInterval: 30,
			},
		}

		assert.Equal(t, "postgres://localhost:2231", cfg.URI)
		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 2231, cfg.Port)
		assert.Equal(t, "postgres", cfg.Username)
		assert.Equal(t, "secret", cfg.Password)
		assert.Equal(t, "testdb", cfg.DBName)
		assert.Equal(t, "require", cfg.SSLMode)
		assert.Equal(t, "test-app", cfg.AppName)
		assert.Equal(t, 5, cfg.ConnTimeout)
		assert.Equal(t, "pool", cfg.Mode)
		assert.Equal(t, 10, cfg.Pool.MaxConns)
		assert.Equal(t, 2, cfg.Pool.MinConns)
		assert.Equal(t, 300, cfg.Pool.MaxConnLifetime)
		assert.Equal(t, 60, cfg.Pool.MaxConnIdleTime)
		assert.Equal(t, 30, cfg.Pool.HealthCheckInterval)
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
			Mongodb: &manifest.MongodbConfig{
				URI:      "mongodb://localhost:27017",
				Username: "user",
				Password: "pass",
				DBName:   "testdb",
				Hosts:    []string{"localhost:27017"},
				Options: manifest.MongodbOptions{
					ReplicaSet: "rs0",
				},
			},
			Postgres: &manifest.PostgresConfig{
				URI: "localhost:2213",
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

		require.NotNil(t, cfg.Mongodb)
		assert.Equal(t, "mongodb://localhost:27017", cfg.Mongodb.URI)
		assert.Equal(t, "user", cfg.Mongodb.Username)

		require.NotNil(t, cfg.Postgres)
		assert.Equal(t, "localhost:2213", cfg.Postgres.URI)

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
		assert.Nil(t, cfg.Mongodb)
		assert.Nil(t, cfg.RabbitMQ)
		assert.Nil(t, cfg.Prometheus)
		assert.Nil(t, cfg.GRPC)
		assert.Nil(t, cfg.HTTP)
	})
}
