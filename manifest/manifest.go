package manifest

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueload "cuelang.org/go/cue/load"
)

//go:embed cue
var cueFS embed.FS

func MustNew(configPath, envPath string) *Config {
	config, err := Build(configPath, envPath)
	if err != nil {
		panic(err)
	}
	return config
}

func New(configPath, envPath string) (*Config, error) {
	return Build(configPath, envPath)
}

func Build(configPath, envPath string) (*Config, error) {
	if _, err := os.Stat(envPath); err == nil {
		LoadEnvFile(envPath)
	}
	cuectx := cuecontext.New()

	overlay, err := getOverlay(cueFS)
	if err != nil {
		return nil, err
	}
	ins := cueload.Instances([]string{"."}, &cueload.Config{
		Dir:     "./cue/",
		Overlay: overlay,
	})
	defaultIns := cuectx.BuildInstance(ins[0])
	if defaultIns.Err() != nil {
		return nil, defaultIns.Err()
	}

	ins2 := cueload.Instances([]string{configPath}, nil)
	userIns := cuectx.BuildInstance(ins2[0])
	if userIns.Err() != nil {
		return nil, userIns.Err()
	}

	unifiedIns := defaultIns.Unify(userIns)
	if unifiedIns.Err() != nil {
		return nil, unifiedIns.Err()
	}

	serviceVal := unifiedIns.LookupPath(cue.ParsePath("service"))
	if !serviceVal.Exists() {
		return nil, errors.New("service field not found")
	}

	var dec Config
	if err := serviceVal.Decode(&dec); err != nil {
		return nil, err
	}

	if err := LoadEnvVars(&dec); err != nil {
		return nil, err
	}

	return &dec, nil
}

func getOverlay(fsys fs.FS) (map[string]cueload.Source, error) {
	overlay := map[string]cueload.Source{}
	cwd, err := os.Getwd()
	if err != nil {
		return overlay, err
	}
	if err := fs.WalkDir(
		fsys, ".",
		func(filename string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			if strings.HasSuffix(filename, ".cue") {
				data, err := fs.ReadFile(fsys, filename)
				if err != nil {
					return err
				}
				path := filepath.Join(cwd, filename)
				overlay[path] = cueload.FromBytes(data)
			}
			return nil
		},
	); err != nil {
		return overlay, fmt.Errorf("walkdir: %v", err)
	}
	return overlay, nil
}
