package config

import (
	"errors"
	"os"

	"github.com/joho/godotenv"
)

func readEnvFile(path string) (map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	vars, err := godotenv.Read(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return vars, err
}

// lookupEnv resolves name with process-environment presence taking precedence
// over the env file. A present-but-empty value is a value, not an absence, so
// an explicit empty configuration survives the overlay.
func lookupEnv(name string, fileVars map[string]string) (string, bool) {
	if value, ok := os.LookupEnv(name); ok {
		return value, true
	}
	value, ok := fileVars[name]
	return value, ok
}
