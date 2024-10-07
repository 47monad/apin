package initr

import (
	"context"
	"github.com/47monad/apin/initropts"
	"github.com/47monad/apin/internal/logger"
)

var Zap = AgentFunc[*initropts.ZapLoggerStore, *LoggerShell](initZapLogger)

func initZapLogger(ctx context.Context, b initropts.Builder[*initropts.ZapLoggerStore]) (*LoggerShell, error) {
	_, err := b.Build()
	if err != nil {
		return nil, err
	}
	zp, err := logger.InitZap()
	if err != nil {
		return nil, err
	}
	shell := &LoggerShell{
		Driver: logger.ZAP,
		Logger: zp,
	}
	return shell, nil
}
