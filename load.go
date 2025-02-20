package apin

import (
	"fmt"

	"github.com/47monad/zaal"
)

func InitWithZaal(configPath, envPath string) (*App, error) {
	config, err := zaal.Build(configPath, envPath)
	if err != nil {
		return nil, fmt.Errorf("unable to build config due to: %v \n", err)
	}

	return FromConfig(config), nil
}
