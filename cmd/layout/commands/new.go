package commands

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"

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
		return defaultConfig()
	}
	return cmd.Config
}

func (cmd NewCommand) Execute([]string) error {
	// TODO: load config file
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer cancel()

	// little hack to notify UI that we are done
	go func() {
		<-ctx.Done()
		_ = os.Stdin.Close()
	}()

	return internal.Deploy(ctx, cmd.Args.URL, cmd.Args.Dest)
}

func defaultConfig() string {
	const configFile = ".layoutrc"
	v, err := os.UserConfigDir()
	if err != nil {
		return configFile
	}
	return filepath.Join(v, "layout", configFile)
}
