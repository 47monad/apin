package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	logger *zap.Logger
}

func InitZap() (Logger, error) {
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

func (zl *zapLogger) Warn(message string, fields LogFields) {
	zl.Log(Warn, message, fields)
}

func (zl *zapLogger) Info(message string, fields LogFields) {
	zl.Log(Info, message, fields)
}

func (zl *zapLogger) Log(level LogLevel, message string, fields LogFields) {
	_f := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		_f = append(_f, zap.Any(k, v))
	}
	zl.logger.Log(zapcore.Level(level), message, _f...)
}

func (zl *zapLogger) Error(message string, fields LogFields) {
	zl.Log(Error, message, fields)
}

func (zl *zapLogger) Fatal(message string, fields LogFields) {
	zl.Log(Fatal, message, fields)
}

func (zl *zapLogger) Debug(message string, fields LogFields) {
	zl.Log(Debug, message, fields)
}
