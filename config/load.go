// Package config loads application-owned configuration from JSON or YAML
// files and overlays environment variables declared by env struct tags.
package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load decodes configPath into destination, then overlays values from envPath
// and the process environment. Process environment values take precedence over
// .env values, which take precedence over values in the configuration file.
// destination must be a non-nil pointer.
func Load(configPath, envPath string, destination any) error {
	dst := reflect.ValueOf(destination)
	if !dst.IsValid() || dst.Kind() != reflect.Pointer || dst.IsNil() {
		return fmt.Errorf("load config file %q: destination must be a non-nil pointer", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("load config file %q: %w", configPath, err)
	}
	switch strings.ToLower(filepath.Ext(configPath)) {
	case ".json":
		err = decodeJSON(data, destination)
	case ".yaml", ".yml":
		err = decodeYAML(data, destination)
	default:
		err = fmt.Errorf("unsupported configuration file extension %q", filepath.Ext(configPath))
	}
	if err != nil {
		return fmt.Errorf("load config file %q: %w", configPath, err)
	}

	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("load config file %q: read env file %q: %w", configPath, envPath, err)
	}
	if err := applyEnv(dst.Elem(), "", dst.Elem().Type().Name(), vars); err != nil {
		return fmt.Errorf("load config file %q: %w", configPath, err)
	}
	return nil
}

// MustLoad is Load but panics when loading or overlaying configuration fails.
func MustLoad(configPath, envPath string, destination any) {
	if err := Load(configPath, envPath, destination); err != nil {
		panic(err)
	}
}

func decodeJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("decode JSON: trailing value")
		}
		return fmt.Errorf("decode JSON trailing content: %w", err)
	}
	return nil
}

func decodeYAML(data []byte, destination any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode YAML: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return fmt.Errorf("decode YAML trailing content: %w", err)
		}
		return errors.New("decode YAML: multiple documents are not allowed")
	}
	return nil
}

func applyEnv(value reflect.Value, prefix, path string, vars map[string]string) error {
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			fieldValue := value.Field(i)
			if field.PkgPath != "" || !fieldValue.CanSet() {
				continue
			}
			fieldPath := path + "." + field.Name
			if tag := field.Tag.Get("env"); tag != "" && tag != "-" {
				key := strings.ToUpper(tag)
				if prefix != "" {
					key = prefix + "_" + key
				}
				if raw, ok := lookupEnv(key, vars); ok {
					if err := setEnvValue(fieldValue, raw); err != nil {
						return fmt.Errorf("environment variable %s for field %s: %w", key, fieldPath, err)
					}
				}
			}
			if err := applyEnv(fieldValue, prefix, fieldPath, vars); err != nil {
				return err
			}
		}
	case reflect.Pointer:
		if value.IsNil() {
			candidate := reflect.New(value.Type().Elem())
			if !containsEnvValue(candidate.Elem(), prefix, vars) {
				return nil
			}
			if err := applyEnv(candidate.Elem(), prefix, path, vars); err != nil {
				return err
			}
			value.Set(candidate)
			return nil
		}
		return applyEnv(value.Elem(), prefix, path, vars)
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		iter := value.MapRange()
		for iter.Next() {
			key := iter.Key()
			mapValue := reflect.New(value.Type().Elem()).Elem()
			mapValue.Set(iter.Value())
			mapPrefix := prefix
			if key.Kind() == reflect.String {
				mapKey := strings.ToUpper(key.String())
				if mapPrefix != "" {
					mapKey = mapPrefix + "_" + mapKey
				}
				mapPrefix = mapKey
			}
			mapPath := fmt.Sprintf("%s[%v]", path, key.Interface())
			if err := applyEnv(mapValue, mapPrefix, mapPath, vars); err != nil {
				return err
			}
			value.SetMapIndex(key, mapValue)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := applyEnv(value.Index(i), prefix, fmt.Sprintf("%s[%d]", path, i), vars); err != nil {
				return err
			}
		}
	}
	return nil
}

func setEnvValue(field reflect.Value, raw string) error {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setEnvValue(field.Elem(), raw)
	}
	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)
	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return err
		}
		field.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(value)
	default:
		return fmt.Errorf("unsupported environment field type %s", field.Type())
	}
	return nil
}

func containsEnvValue(value reflect.Value, prefix string, vars map[string]string) bool {
	switch value.Kind() {
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("env")
			if tag == "-" {
				tag = ""
			}
			key := strings.ToUpper(tag)
			if prefix != "" && key != "" {
				key = prefix + "_" + key
			}
			if key != "" {
				if _, ok := lookupEnv(key, vars); ok {
					return true
				}
			}
			if containsEnvValue(value.Field(i), prefix, vars) {
				return true
			}
		}
	case reflect.Pointer:
		if value.IsNil() {
			return containsEnvType(value.Type().Elem(), prefix, vars)
		}
		return containsEnvValue(value.Elem(), prefix, vars)
	case reflect.Map:
		iter := value.MapRange()
		for iter.Next() {
			mapPrefix := prefix
			key := iter.Key()
			if key.Kind() == reflect.String {
				mapKey := strings.ToUpper(key.String())
				if mapPrefix != "" {
					mapKey = mapPrefix + "_" + mapKey
				}
				mapPrefix = mapKey
			}
			if containsEnvValue(iter.Value(), mapPrefix, vars) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if containsEnvValue(value.Index(i), prefix, vars) {
				return true
			}
		}
	}
	return false
}

func containsEnvType(typ reflect.Type, prefix string, vars map[string]string) bool {
	switch typ.Kind() {
	case reflect.Struct:
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			if field.PkgPath != "" {
				continue
			}
			tag := field.Tag.Get("env")
			if tag == "-" {
				tag = ""
			}
			key := strings.ToUpper(tag)
			if prefix != "" && key != "" {
				key = prefix + "_" + key
			}
			if key != "" {
				if _, ok := lookupEnv(key, vars); ok {
					return true
				}
			}
			if containsEnvType(field.Type, prefix, vars) {
				return true
			}
		}
	case reflect.Pointer:
		return containsEnvType(typ.Elem(), prefix, vars)
	}
	return false
}
