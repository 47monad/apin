package initr

import "github.com/47monad/apin/internal/logger"

type LoggerShell struct {
	Logger logger.Logger
	Driver logger.LogDriver
}
