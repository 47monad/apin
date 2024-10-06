package apin

import (
	"errors"
	"github.com/47monad/apin/initr"
	"github.com/47monad/apin/internal/logger"
)

func Init(driver logger.LogDriver) (logger.Logger, error) {
	var (
		_lg logger.Logger
		err error
	)
	if driver == logger.ZAP {
		_lg, err = initr.InitZapLogger()
		if err != nil {
			return nil, err
		}
	}
	if _lg == nil {
		return nil, errors.New("invalid logger driver")
	}
	return _lg, nil
}
