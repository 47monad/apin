package initr

import (
	"github.com/47monad/apin/internal/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
}

func InitZapLogger() (logger.Logger, error) {
	zl := &zapLogger{}
	err := zl.init()
	if err != nil {
		return nil, err
	}
	return zl, nil
}

func (zl *zapLogger) init() error {
	config := zap.NewProductionConfig()
	encoderConfig := zap.NewProductionEncoderConfig()
	zapcore.TimeEncoderOfLayout("Jan _2 15:04:05.000000000")
	encoderConfig.StacktraceKey = "" // to hide stacktrace info
	config.EncoderConfig = encoderConfig

	zapLog, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		return err
	}
	zl.logger = zapLog
	return nil
}

func (zl *zapLogger) Warn(message string, fields logger.LogFields) {
	zl.Log(logger.Warn, message, fields)
}

func (zl *zapLogger) Info(message string, fields logger.LogFields) {
	zl.Log(logger.Info, message, fields)
}

func (zl *zapLogger) Log(level logger.LogLevel, message string, fields logger.LogFields) {
	_f := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		_f = append(_f, zap.Any(k, v))
	}
	zl.logger.Log(zapcore.Level(level), message, _f...)
}

func (zl *zapLogger) Error(message string, fields logger.LogFields) {
	zl.Log(logger.Error, message, fields)
}

func (zl *zapLogger) Fatal(message string, fields logger.LogFields) {
	zl.Log(logger.Fatal, message, fields)
}

func (zl *zapLogger) Debug(message string, fields logger.LogFields) {
	zl.Log(logger.Debug, message, fields)
}
