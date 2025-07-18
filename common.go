package apin

import "github.com/go-logr/logr"

type LoggerShell struct {
	Logger logr.Logger
}

type Builder[K any] interface {
	Build() (K, error)
}
