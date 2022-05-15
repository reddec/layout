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
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
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
	const configFile = "layout.yaml"
	v, err := os.UserConfigDir()
	if err != nil {
		return configFile
	}
	return filepath.Join(v, "layout", configFile)
}
