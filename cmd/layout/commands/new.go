package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"layout/internal"
)

type NewCommand struct {
	Config string `short:"c" long:"config" env:"CONFIG" description:"Path to configuration file, use show config command to locate default location"`
	Args   struct {
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
	})
}
