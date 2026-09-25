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

// EnvVars holds environment values parsed from a .env file. Values are kept
// out of the process environment: a Config is overlaid from an explicit set of
// variables, so concurrent loads cannot race on the global environment and one
// load cannot leak variables into the next.
//
// Use [ParseEnvFile] to build one. The zero value is usable and empty.
type EnvVars map[string]string

// ParseEnvFile reads a .env file into an isolated variable set. It applies the
// same parsing rules as [LoadEnvFile] but does not modify the process
// environment, so the returned values are visible only to the Config they are
// overlaid onto.
func ParseEnvFile(path string) (EnvVars, error) {
	vars, err := godotenv.Read(path)
	if err != nil {
		return nil, err
	}
	return EnvVars(vars), nil
}

// LoadEnvFile loads path into the process environment.
//
// Deprecated: it mutates global state, which makes concurrent builds race and
// leaks variables into every later build in the process. [Build] no longer
// uses it, and neither should new code — use [ParseEnvFile] instead, which
// returns the same values without touching the environment.
func LoadEnvFile(path string) error {
	err := godotenv.Load(path)
	if err != nil {
		return err
	}
	return nil
}

// Lookup returns the value of the environment variable name. Variables already
// set in the process environment win over the parsed file, matching the
// historical behaviour of loading the file into the environment (godotenv
// never overrides an existing variable). Empty values count as unset.
func (e EnvVars) Lookup(name string) (string, bool) {
	if value := os.Getenv(name); value != "" {
		return value, true
	}
	value, ok := e[name]
	if !ok || value == "" {
		return "", false
	}
	return value, true
}

// hasPrefix reports whether any variable visible to the overlay — the process
// environment or the parsed file — starts with prefix.
func (e EnvVars) hasPrefix(prefix string) bool {
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	for name := range e {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Overlay applies the variables to cfg. The process environment takes
// precedence over the parsed file, and both take precedence over the values
// decoded from the manifest. The process environment is only read, never
// written, so an Overlay is safe to run concurrently with anything else.
func (e EnvVars) Overlay(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is nil")
	}
	ensureOptionalSections(cfg, e)

	val := reflect.ValueOf(cfg).Elem()
	if err := setFields(val, "", e.Lookup); err != nil {
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

// LoadEnvVars overlays cfg from the process environment alone. Use
// [EnvVars.Overlay] with a set from [ParseEnvFile] to include a .env file.
func LoadEnvVars(cfg *Config) error {
	return EnvVars(nil).Overlay(cfg)
}

// ensureOptionalSections allocates optional config sections that are
// absent from the CUE file when environment variables matching their
// prefix are set. Currently only the postgres section supports this.
func ensureOptionalSections(cfg *Config, env EnvVars) {
	if cfg.Postgres != nil {
		return
	}
	if env.hasPrefix("POSTGRES_") {
		cfg.Postgres = &PostgresConfig{}
	}
}

func setFields(val reflect.Value, ctx string, lookup func(string) (string, bool)) error {
	for i := 0; i < val.Type().NumField(); i++ {
		field := val.Type().Field(i)
		fieldVal := val.Field(i)

		envTag := field.Tag.Get("env")
		if envTag != "" {
			if ctx != "" {
				envTag = ctx + "_" + envTag
			}
			tagName := strings.ToUpper(envTag)
			envValue, ok := lookup(tagName)
			if !ok {
				// Fall back to a deprecated name if one is declared, so
				// existing deployments keep working after a rename.
				if depTag := field.Tag.Get("envDeprecated"); depTag != "" {
					depName := strings.ToUpper(depTag)
					if depValue, found := lookup(depName); found {
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
				if err := setFields(fieldVal, "", lookup); err != nil {
					return err
				}
			}
		} else if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() &&
			fieldVal.Elem().Kind() == reflect.Struct {
			if err := setFields(fieldVal.Elem(), "", lookup); err != nil {
				return err
			}
		} else if fieldVal.Kind() == reflect.Map {
			keys := fieldVal.MapKeys()

			for _, k := range keys {
				origValue := fieldVal.MapIndex(k)
				newValue := reflect.New(origValue.Type()).Elem()
				newValue.Set(origValue)

				if newValue.Kind() == reflect.Struct {
					if err := setFields(newValue, k.String(), lookup); err != nil {
						return err
					}
					fieldVal.SetMapIndex(k, newValue)
				}

			}
		}
	}
	return nil
}
