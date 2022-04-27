package commands

import "fmt"

type ShowCommand struct {
	Config ShowConfigCommand `command:"config" description:"location of default config file"`
}

type ShowConfigCommand struct {
}

func (cmd ShowConfigCommand) Execute([]string) error {
	_, err := fmt.Println(defaultConfigFile())
	return err
}
