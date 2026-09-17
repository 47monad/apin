package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func WriteToJson(c *Config, outputPath string) error {
	jsonData, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	dirPath := filepath.Dir(outputPath)
	if dirPath != "." && dirPath != "/" {
		os.MkdirAll(dirPath, os.ModePerm)
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return err
	}
	return nil
}
