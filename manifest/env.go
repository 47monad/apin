package manifest

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnvFile(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}
	return nil
}

func LoadEnvVars(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	ensureOptionalSections(cfg)

	val := reflect.ValueOf(cfg).Elem()
	if err := setFields(val, ""); err != nil {
		return err
	}

	// Environment values bypass the CUE schema (see Build), so validate
	// constraints that the schema would otherwise enforce.
	if cfg.Postgres != nil {
		if err := cfg.Postgres.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ensureOptionalSections allocates optional config sections that are
// absent from the CUE file when environment variables matching their
// prefix are set. Currently only the postgres section supports this.
func ensureOptionalSections(cfg *Config) {
	if cfg.Postgres != nil {
		return
	}
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, "POSTGRES_") {
			cfg.Postgres = &PostgresConfig{}
			return
		}
	}
}

func setFields(val reflect.Value, ctx string) error {
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Type().Field(i)
		fieldVal := val.Field(i)

		envTag := field.Tag.Get("env")
		if envTag != "" {
			if ctx != "" {
				envTag = ctx + "_" + envTag
			}
			tagName := strings.ToUpper(envTag)
			envValue := os.Getenv(tagName)
			if envValue == "" {
				// Fall back to a deprecated name if one is declared, so
				// existing deployments keep working after a rename.
				if depTag := field.Tag.Get("envDeprecated"); depTag != "" {
					depName := strings.ToUpper(depTag)
					if depValue := os.Getenv(depName); depValue != "" {
						fmt.Fprintf(os.Stderr, "warning: environment variable %s is deprecated, use %s instead\n",
							depName, tagName)
						envValue = depValue
					}
				}
			}
			if envValue != "" {
				switch fieldVal.Kind() {
				case reflect.String:
					fieldVal.SetString(envValue)
				case reflect.Int:
					intValue, err := strconv.Atoi(envValue)
					if err != nil {
						fmt.Println(intValue, err)
						return fmt.Errorf("error converting env var %s to int: %w",
							envTag, err)
					}
					fieldVal.SetInt(int64(intValue))
				case reflect.Bool:
					boolValue, err := strconv.ParseBool(envValue)
					if err != nil {
						return fmt.Errorf("error converting env var %s to bool: %w",
							envTag, err)
					}
					fieldVal.SetBool(boolValue)
				}
			}
		} else if fieldVal.Kind() == reflect.Struct {
			// Recursively process nested structs
			if fieldVal.Type().String() != "time.Time" {
				if err := setFields(fieldVal, ""); err != nil {
					return err
				}
			}
		} else if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() &&
			fieldVal.Elem().Kind() == reflect.Struct {
			if err := setFields(fieldVal.Elem(), ""); err != nil {
				return err
			}
		} else if fieldVal.Kind() == reflect.Map {
			keys := fieldVal.MapKeys()

			for _, k := range keys {
				origValue := fieldVal.MapIndex(k)
				newValue := reflect.New(origValue.Type()).Elem()
				newValue.Set(origValue)

				if newValue.Kind() == reflect.Struct {
					if err := setFields(newValue, k.String()); err != nil {
						return err
					}
					fieldVal.SetMapIndex(k, newValue)
				}

			}
		}
	}
	return nil
}
