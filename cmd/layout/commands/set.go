package commands

import (
	"fmt"
)

type SetCommand struct {
	Git     SetGitCommand     `command:"git" description:"git client mode"`
	Default SetDefaultCommand `command:"default" description:"URL pattern to resolve layout"`
}

func (cmd SetCommand) Execute([]string) error {
	_, err := fmt.Println(defaultConfigFile())
	return err
}

type SetGitCommand struct {
	ConfigSource
	Args struct {
		Git gitMode `positional-arg-name:"mode" required:"yes" description:"Git client. Default value as in config file or auto. Supported: auto, native, embedded"`
	} `positional-args:"yes"`
}

func (sc *SetGitCommand) Execute([]string) error {
	cfg, err := sc.readConfig()
	if err != nil {
		return err
	}
	cfg.Git = sc.Args.Git
	return cfg.Save(sc.configFile())
}

type SetDefaultCommand struct {
	ConfigSource
	Args struct {
		Pattern string `positional-arg-name:"pattern" required:"yes" description:"Pattern for requests without abbreviations. May contain {0}"`
	} `positional-args:"yes"`
}

func (sc *SetDefaultCommand) Execute([]string) error {
	cfg, err := sc.readConfig()
	if err != nil {
		return err
	}
	cfg.Default = sc.Args.Pattern
	return cfg.Save(sc.configFile())
}
