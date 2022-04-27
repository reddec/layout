package main

import (
	"fmt"
	"os"

	"layout/cm/layout/commands"

	"github.com/jessevdk/go-flags"
)

//nolint:gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

type Config struct {
	New  commands.NewCommand  `command:"new" description:"deploy layout"`
	Show commands.ShowCommand `command:"show" description:"show configuration"`
}

func main() {
	var config Config
	parser := flags.NewParser(&config, flags.Default)
	parser.ShortDescription = "Scaffold new project based on layout"
	parser.LongDescription = fmt.Sprintf("Scaffold new project based on layout\nlayout %s, commit %s, built at %s by %s\nAuthor: Aleksandr Baryshnikov <owner@reddec.net>", version, commit, date, builtBy)
	parser.EnvNamespace = "LAYOUT"
	if _, err := parser.Parse(); err != nil {
		os.Exit(1)
	}
}
