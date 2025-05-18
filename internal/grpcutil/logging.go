package grpcutil

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"google.golang.org/grpc"
)

func InterceptLogsWith(l logr.Logger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		switch lvl {
		case logging.LevelError:
			l.Error(nil, msg, fields...)
		default:
			l.Info(msg, fields...)
		}
	})
}

func NewLoggingInterceptor(l logr.Logger) grpc.UnaryServerInterceptor {
	loggerOpts := []logging.Option{
		logging.WithLogOnEvents(logging.StartCall, logging.FinishCall),
	}
	return logging.UnaryServerInterceptor(InterceptLogsWith(l), loggerOpts...)
}
