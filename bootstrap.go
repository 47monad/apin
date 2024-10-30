package apin

import (
	"github.com/47monad/sercon"
)

type BootstrapOptions struct {
	ConfigPath string
	EnvPath    string
	UseRelay   bool
}

type BootstrapOption func(*BootstrapOptions)

func Bootstrap(filename string, options ...BootstrapOption) (*App, error) {
	opts := prepareOptions(filename, options)
	conf, err := getConfig(opts)
	if err != nil {
		return nil, err
	}

	app := FromConfig(conf)

	return app, nil
}

func getConfig(opts *BootstrapOptions) (*sercon.Config, error) {
	if opts.UseRelay {
		return sercon.LoadFromRelay(
			sercon.WithConfigPath(opts.ConfigPath),
		)
	}
	return sercon.Load(
		sercon.WithConfigPath(opts.ConfigPath),
		sercon.WithEnvPath(opts.EnvPath),
	)
}

func prepareOptions(filename string, opts []BootstrapOption) *BootstrapOptions {
	newOpts := &BootstrapOptions{
		ConfigPath: "./config/" + filename + ".pkl",
		EnvPath:    "./config/." + filename + ".env",
		UseRelay:   false,
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

func UseRelay() BootstrapOption {
	return func(opts *BootstrapOptions) {
		opts.UseRelay = true
	}
}
