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
// and the process environment. An empty configPath skips file decoding, so
// configuration may be supplied through the environment alone. A missing
// envPath is optional. Process environment values take precedence over values
// from envPath, which take precedence over values in the configuration file.
//
// A present-but-empty environment value is a value, not an absence: it
// overrides lower-precedence sources, becomes an empty string or empty string
// slice, and reports a contextual conversion error for scalar fields.
// destination must be a non-nil pointer.
func Load(configPath, envPath string, destination any) error {
	where := loadContext(configPath)
	dst := reflect.ValueOf(destination)
	if !dst.IsValid() || dst.Kind() != reflect.Pointer || dst.IsNil() {
		return fmt.Errorf("%s: destination must be a non-nil pointer", where)
	}

	if configPath != "" {
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
	}

	vars, err := readEnvFile(envPath)
	if err != nil {
		return fmt.Errorf("%s: read env file %q: %w", where, envPath, err)
	}
	overlay := newEnvOverlay(vars)
	if err := applyEnv(dst.Elem(), "", dst.Elem().Type().Name(), overlay); err != nil {
		return fmt.Errorf("%s: %w", where, err)
	}
	return nil
}

// loadContext names the source of a loading error. Environment-only loading has
// no configuration file, so the message does not fabricate an empty path.
func loadContext(configPath string) string {
	if configPath == "" {
		return "load config"
	}
	return fmt.Sprintf("load config file %q", configPath)
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

// envScope identifies one position in the destination: a type reached under an
// environment-variable prefix. Two visits to the same scope on one path mean
// the destination type is recursive and no map key separated the visits, so
// descending again cannot make progress.
type envScope struct {
	typ    reflect.Type
	prefix string
}

// envOverlay carries the state of one environment overlay. Nil pointers are the
// only structure an overlay invents, so the scopes it is currently
// materializing are tracked to keep a recursive type finite.
type envOverlay struct {
	vars     map[string]string
	visited  map[envScope]bool
	inFlight map[envScope]bool
}

func newEnvOverlay(vars map[string]string) *envOverlay {
	return &envOverlay{
		vars:     vars,
		visited:  make(map[envScope]bool),
		inFlight: make(map[envScope]bool),
	}
}

// enter records scope as being visited on the current path and reports whether
// the caller may descend into it. The returned function must be called when the
// visit ends, so a later sibling of the same type is still visited.
func (o *envOverlay) enter(scope envScope) (bool, func()) {
	if o.visited[scope] {
		return false, func() {}
	}
	o.visited[scope] = true
	return true, func() { delete(o.visited, scope) }
}

func applyEnv(value reflect.Value, prefix, path string, overlay *envOverlay) error {
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
				if raw, ok := lookupEnv(key, overlay.vars); ok {
					if err := setEnvValue(fieldValue, raw); err != nil {
						return fmt.Errorf("environment variable %s for field %s: %w", key, fieldPath, err)
					}
				}
			}
			if err := applyEnv(fieldValue, prefix, fieldPath, overlay); err != nil {
				return err
			}
		}
	case reflect.Pointer:
		if value.IsNil() {
			// A nil pointer is materialized only when the environment supplies a
			// value beneath it. With this exact scope already in flight, a
			// recursive type would chain instances forever, so the link stays
			// nil rather than growing without a concrete instance to justify it.
			scope := envScope{typ: value.Type(), prefix: prefix}
			if overlay.inFlight[scope] {
				return nil
			}
			candidate := reflect.New(value.Type().Elem())
			if !containsEnvValue(candidate.Elem(), prefix, overlay) {
				return nil
			}
			overlay.inFlight[scope] = true
			defer delete(overlay.inFlight, scope)
			if err := applyEnv(candidate.Elem(), prefix, path, overlay); err != nil {
				return err
			}
			value.Set(candidate)
			return nil
		}
		return applyEnv(value.Elem(), prefix, path, overlay)
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
			if err := applyEnv(mapValue, mapPrefix, mapPath, overlay); err != nil {
				return err
			}
			value.SetMapIndex(key, mapValue)
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := applyEnv(value.Index(i), prefix, fmt.Sprintf("%s[%d]", path, i), overlay); err != nil {
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
	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported environment field type %s", field.Type())
		}
		entries := splitList(raw)
		slice := reflect.MakeSlice(field.Type(), len(entries), len(entries))
		for i, entry := range entries {
			slice.Index(i).SetString(entry)
		}
		field.Set(slice)
	default:
		return fmt.Errorf("unsupported environment field type %s", field.Type())
	}
	return nil
}

// splitList decodes a comma-separated environment value into trimmed entries.
// Blanks between separators are preserved so consumers can validate them. An
// empty or whitespace-only value decodes to an empty list.
func splitList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return []string{}
	}
	parts := strings.Split(trimmed, ",")
	entries := make([]string, len(parts))
	for i, part := range parts {
		entries[i] = strings.TrimSpace(part)
	}
	return entries
}

func containsEnvValue(value reflect.Value, prefix string, overlay *envOverlay) bool {
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
				if _, ok := lookupEnv(key, overlay.vars); ok {
					return true
				}
			}
			if containsEnvValue(value.Field(i), prefix, overlay) {
				return true
			}
		}
	case reflect.Pointer:
		if value.IsNil() {
			return containsEnvType(value.Type().Elem(), prefix, overlay)
		}
		return containsEnvValue(value.Elem(), prefix, overlay)
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
			if containsEnvValue(iter.Value(), mapPrefix, overlay) {
				return true
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if containsEnvValue(value.Index(i), prefix, overlay) {
				return true
			}
		}
	}
	return false
}

func containsEnvType(typ reflect.Type, prefix string, overlay *envOverlay) bool {
	// A recursive destination type reaches itself again through the pointer
	// case below. Stop when the path loops back to a scope already being
	// scanned: its fields were searched for this prefix on the first visit.
	scope := envScope{typ: typ, prefix: prefix}
	proceed, leave := overlay.enter(scope)
	if !proceed {
		return false
	}
	defer leave()

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
				if _, ok := lookupEnv(key, overlay.vars); ok {
					return true
				}
			}
			if containsEnvType(field.Type, prefix, overlay) {
				return true
			}
		}
	case reflect.Pointer:
		return containsEnvType(typ.Elem(), prefix, overlay)
	}
	return false
}
