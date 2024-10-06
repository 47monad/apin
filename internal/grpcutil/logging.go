package grpcutil

import (
	"context"
	"fmt"
	"github.com/47monad/apin/internal/logger"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

func InterceptLogsWith(l logger.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		f := make(logger.LogFields, len(fields)/2)

		for i := 0; i < len(fields); i += 2 {
			key := fields[i]
			value := fields[i+1]
			f[key.(string)] = value
		}

		switch lvl {
		case logging.LevelDebug:
			l.Debug(msg, f)
		case logging.LevelInfo:
			l.Info(msg, f)
		case logging.LevelWarn:
			l.Warn(msg, f)
		case logging.LevelError:
			l.Error(msg, f)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

func NewLoggingInterceptor(l logger.Logger) grpc.UnaryServerInterceptor {
	loggerOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}
	return logging.UnaryServerInterceptor(InterceptLogsWith(l), loggerOpts...)
}
