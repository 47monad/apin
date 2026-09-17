package apin

import (
	"github.com/47monad/apin/manifest"
)

// LoadConfig reads the service manifest from a CUE file, validates it
// against the built-in schema, and applies environment variable overrides
// from the given env file. It is the user-facing entry point for
// configuration; initrs take the returned config sections via WithConfig.
func LoadConfig(configPath, envPath string) (*manifest.Config, error) {
	return manifest.New(configPath, envPath)
}

// MustLoadConfig is LoadConfig but panics on failure. Useful for
// small programs and examples where a bad manifest should abort startup.
func MustLoadConfig(configPath, envPath string) *manifest.Config {
	return manifest.MustNew(configPath, envPath)
}
