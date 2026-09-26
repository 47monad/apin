package manifest_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/47monad/apin/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZaal(t *testing.T) {
	clearEnv(t)

	res, err := manifest.Build(
		"./testdata/main.cue",
		"./testdata/main.env",
	)
	if err != nil {
		t.Fatal(err)
	}

	js, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(js))
}

func TestNew(t *testing.T) {
	// The overlay reads the process environment, so a stray variable in the
	// developer's shell would change the outcome — including panicking the
	// success case below when it is a manifest environment value.
	clearEnv(t)

	tests := []struct {
		name       string
		configPath string
		envPath    string
		wantName   string
		wantErr    bool
	}{
		{
			name:       "success",
			configPath: "./testdata/main.cue",
			envPath:    "./testdata/main.env",
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
			cfg, err := manifest.New(tt.configPath, tt.envPath)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, cfg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, cfg)
			assert.Equal(t, tt.wantName, cfg.Name)
		})
	}
}

func TestMustNew(t *testing.T) {
	clearEnv(t)

	t.Run("success", func(t *testing.T) {
		cfg := manifest.MustNew("./testdata/main.cue", "./testdata/main.env")
		require.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Name)
	})
	t.Run("bad_manifest_path_panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("MustNew() did not panic on bad manifest path")
			}
		}()
		_ = manifest.MustNew("nonexistent.cue", "nonexistent.env")
	})
}

func TestBuildLoadEnvFileError(t *testing.T) {
	clearEnv(t)

	t.Run("malformed_env/error", func(t *testing.T) {
		dir := t.TempDir()
		path := dir + "/bad.env"
		require.NoError(t, os.WriteFile(path, []byte("FOO=\"unterminated\n"), 0o644))

		cfg, err := manifest.Build("./testdata/main.cue", path)
		require.Error(t, err)
		require.Nil(t, cfg)
		assert.Contains(t, err.Error(), "load env file")
	})

	t.Run("env_path_is_directory/error", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := manifest.Build("./testdata/main.cue", dir)
		require.Error(t, err)
		require.Nil(t, cfg)
		assert.Contains(t, err.Error(), "load env file")
	})

	t.Run("missing_env/ok", func(t *testing.T) {
		cfg, err := manifest.Build("./testdata/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.Equal(t, "test", cfg.Name)
	})
}

func TestBuildEnvironmentPrecedence(t *testing.T) {
	clearEnv(t)
	dir := t.TempDir()
	configPath := dir + "/service.cue"
	envPath := dir + "/service.env"
	require.NoError(t, os.WriteFile(configPath, []byte(`service: {name: "precedence", host: "from-config"}`), 0o600))
	require.NoError(t, os.WriteFile(envPath, []byte("HOST=from-file\n"), 0o600))

	fileConfig, err := manifest.Build(configPath, envPath)
	require.NoError(t, err)
	assert.Equal(t, "from-file", fileConfig.Host)

	t.Setenv("HOST", "from-process")
	processConfig, err := manifest.Build(configPath, envPath)
	require.NoError(t, err)
	assert.Equal(t, "from-process", processConfig.Host)
}

func TestGRPCClientAddressDefault(t *testing.T) {
	// Shield the fixture from client-address variables the developer's own
	// environment may export, so the empty CUE default is what we observe.
	clearEnv(t)

	t.Run("empty_client_uses_default_address/ok", func(t *testing.T) {
		cfg, err := manifest.Build("./testdata/grpc_client_default/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.GRPC)
		client, ok := cfg.GRPC.Clients["uwcl"]
		require.True(t, ok)
		assert.Equal(t, "", client.Address)
	})

	t.Run("env_overrides_default_address/ok", func(t *testing.T) {
		t.Setenv("UWCL_GRPC_CLIENT_ADDRESS", "localhost:50051")
		cfg, err := manifest.Build("./testdata/grpc_client_default/main.cue", "nonexistent.env")
		require.NoError(t, err)
		require.NotNil(t, cfg.GRPC)
		assert.Equal(t, "localhost:50051", cfg.GRPC.Clients["uwcl"].Address)
	})
}
