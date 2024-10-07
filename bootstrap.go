package apin

import (
	"github.com/47monad/sercon"
)

type BootstrapOptions struct {
	ConfigPath string
	EnvPath    string
}

type BootstrapOption func(*BootstrapOptions)

func Bootstrap(filename string, options ...BootstrapOption) (*App, error) {
	opts := prepareOptions(filename, options)
	conf, err := sercon.Load(
		sercon.WithConfigPath(opts.ConfigPath),
		sercon.WithEnvPath(opts.EnvPath),
	)
	if err != nil {
		return nil, err
	}

	app := FromConfig(conf)

	return app, nil
}

func prepareOptions(filename string, opts []BootstrapOption) *BootstrapOptions {
	newOpts := &BootstrapOptions{
		ConfigPath: "./config/" + filename + ".pkl",
		EnvPath:    "./config/." + filename + ".env",
	}

	for _, opt := range opts {
		opt(newOpts)
	}

	return newOpts
}

func WithConfigPath(path string) BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.ConfigPath = path
	}
}

func WithEnvPath(path string) BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.EnvPath = path
	}
}
