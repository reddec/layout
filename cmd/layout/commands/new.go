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
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/reddec/layout/internal"
	"github.com/reddec/layout/internal/ui"
	"github.com/reddec/layout/internal/ui/nice"
	"github.com/reddec/layout/internal/ui/simple"
)

type NewCommand struct {
	Version string `long:"version" env:"VERSION" description:"Override binary version to bypass manifest restriction"`
	Config  string `short:"c" long:"config" env:"CONFIG" description:"Path to configuration file, use show config command to locate default location"`
	UI      string `short:"u" long:"ui" env:"UI" description:"UI mode" default:"nice" choice:"nice" choice:"simple"`
	Debug   bool   `short:"d" long:"debug" env:"DEBUG" description:"Enable debug mode"`
	AskOnce bool   `short:"a" long:"ask-once" env:"ASK_ONCE" description:"Do not retry on wrong user input, good for automation"`
	Args    struct {
		URL  string `positional-arg-name:"source" required:"yes" description:"URL, abbreviation or path to layout"`
		Dest string `positional-arg-name:"destination" required:"yes" description:"Destination directory, will be created"`
	} `positional-args:"yes"`
}

func (cmd NewCommand) configFile() string {
	if cmd.Config == "" {
		return defaultConfigFile()
	}
	return cmd.Config
}

func (cmd NewCommand) Execute([]string) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	config, err := LoadConfig(cmd.configFile())
	if err != nil {
		return fmt.Errorf("read config %s: %w", cmd.configFile(), err)
	}

	var display ui.UI = simple.Default()
	switch cmd.UI {
	case "nice":
		display = nice.New()
	}
	// little hack to notify UI that we are done
	go func() {
		<-ctx.Done()
		_ = os.Stdin.Close()
	}()

	return internal.Deploy(ctx, internal.Config{
		Source:  cmd.Args.URL,
		Target:  cmd.Args.Dest,
		Aliases: config.Abbreviations,
		Default: config.Default,
		Display: display,
		Debug:   cmd.Debug,
		Version: cmd.Version,
		AskOnce: cmd.AskOnce,
	})
}
