package logger

type LogDriver int
type LogLevel int
type LogFields = map[string]interface{}

const (
	ZAP LogDriver = iota
)

const (
	Debug LogLevel = iota - 1
	Info
	Warn
	Error
	DPanic
	Panic
	Fatal
)

type Logger interface {
	Info(message string, fields LogFields)
	Log(level LogLevel, message string, fields LogFields)
	Error(message string, fields LogFields)
	Warn(message string, fields LogFields)
	Fatal(message string, fields LogFields)
	Debug(message string, fields LogFields)
}
