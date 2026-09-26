package config

import (
	"os"

	"github.com/joho/godotenv"
)

func readEnvFile(path string) (map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	return godotenv.Read(path)
}

func lookupEnv(name string, fileVars map[string]string) (string, bool) {
	if value, ok := os.LookupEnv(name); ok && value != "" {
		return value, true
	}
	value, ok := fileVars[name]
	return value, ok && value != ""
}
