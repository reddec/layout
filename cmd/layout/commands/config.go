/*
Copyright 2022 Aleksandr Baryshnikov

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package commands

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Default       string                 `yaml:"default,omitempty"`       // pattern for requests without abbreviations
	Abbreviations map[string]string      `yaml:"abbreviations,omitempty"` // abbreviations, (ex: alias:owner/repo), stored as alias => pattern ({0} as placeholder)
	Values        map[string]interface{} `yaml:"values,omitempty"`        // global default values
	Git           gitMode                `yaml:"git,omitempty"`           // global default (if not defined by flag) git mode: auto (default), native, embedded
}

func LoadConfig(file string) (*Config, error) {
	var config = Config{
		Git: "auto", // default
	}
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
	const configFile = "layout.yaml"
	v, err := os.UserConfigDir()
	if err != nil {
		return configFile
	}
	return filepath.Join(v, "layout", configFile)
}

func (cfg *Config) Save(file string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config")
	}
	return ioutil.WriteFile(file, data, 0700)
}

type ConfigSource struct {
	Config string `short:"c" long:"config" env:"CONFIG" description:"Path to configuration file, use show config command to locate default location"`
}

func (cmd ConfigSource) configFile() string {
	if cmd.Config == "" {
		return defaultConfigFile()
	}
	return cmd.Config
}

func (cmd ConfigSource) readConfig() (*Config, error) {
	return LoadConfig(cmd.configFile())
}

type gitMode string

var (
	gitAuto     gitMode = "auto"
	gitNative   gitMode = "native"
	gitEmbedded gitMode = "embedded"
)

func (g *gitMode) UnmarshalText(text []byte) error {
	v := string(text)
	switch v {
	case "auto":
		*g = gitAuto
	case "native":
		*g = gitNative
	case "embedded":
		*g = gitEmbedded
	default:
		return errors.New("unknown git mode " + v)
	}
	return nil
}

func (g *gitMode) UnmarshalFlag(value string) error {
	return g.UnmarshalText([]byte(value))
}

func (g *gitMode) UnmarshalYAML(value *yaml.Node) error {
	*g = gitMode(value.Value)
	return nil
}
