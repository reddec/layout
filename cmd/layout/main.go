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

package main

import (
	"fmt"
	"os"

	"github.com/reddec/layout/cmd/layout/commands"

	"github.com/jessevdk/go-flags"
)

//nolint:gochecknoglobals
var (
	version = ""
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type Config struct {
	New  commands.NewCommand  `command:"new" description:"deploy layout"`
	Show commands.ShowCommand `command:"show" description:"show configuration"`
	Set  commands.SetCommand  `command:"set" description:"set configuration"`
}

func main() {
	var config Config
	config.New.Version = version
	parser := flags.NewParser(&config, flags.Default)
	parser.ShortDescription = "Create new project based on layout"
	parser.LongDescription = fmt.Sprintf("Create new project based on layout\nlayout %s, commit %s, built at %s by %s\nAuthor: Aleksandr Baryshnikov <owner@reddec.net>", version, commit, date, builtBy)
	parser.EnvNamespace = "LAYOUT"
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
