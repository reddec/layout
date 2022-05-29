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
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type ShowCommand struct {
	ConfigFile ShowConfigFileCommand `command:"config-file" description:"location of default config file"`
	Config     ShowConfigCommand     `command:"config" description:"current config"`
}

type ShowConfigFileCommand struct {
}

func (cmd ShowConfigFileCommand) Execute([]string) error {
	_, err := fmt.Println(defaultConfigFile())
	return err
}

type ShowConfigCommand struct {
	ConfigSource
}

func (cmd ShowConfigCommand) Execute([]string) error {
	c, err := cmd.readConfig()
	if err != nil {
		return err
	}
	return yaml.NewEncoder(os.Stdout).Encode(c)
}
