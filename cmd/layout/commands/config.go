package commands

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Default       string            // pattern for requests without abbreviations
	Abbreviations map[string]string // abbreviations, (ex: alias:owner/repo), stored as alias => pattern ({0} as placeholder)
}

func LoadConfig(file string) (*Config, error) {
	var config Config
	f, err := os.Open(file)
	if errors.Is(err, os.ErrNotExist) {
		return &config, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return &config, yaml.NewDecoder(f).Decode(&config)
}

func defaultConfigFile() string {
	const configFile = ".layoutrc"
	v, err := os.UserConfigDir()
	if err != nil {
		return configFile
	}
	return filepath.Join(v, "layout", configFile)
}
