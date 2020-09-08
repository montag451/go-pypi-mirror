package cmd

import (
	"flag"
	"fmt"
	"os"
)

type command interface {
	FlagSet() *flag.FlagSet
	Execute() error
}

var commands = map[string]command{}

func registerCommand(cmd command) {
	name := cmd.FlagSet().Name()
	if _, ok := commands[name]; ok {
		panic(fmt.Sprintf("command %q already registered", name))
	}
	commands[name] = cmd
}

func Execute() error {
	if len(os.Args) <= 1 {
		fmt.Println("available commands:")
		for cmd, _ := range commands {
			fmt.Println(cmd)
		}
		os.Exit(1)
	}
	name := os.Args[1]
	cmd, ok := commands[name]
	if !ok {
		return fmt.Errorf("unknown command %q", name)
	}
	flags := cmd.FlagSet()
	flags.Parse(os.Args[2:])
	if err := cmd.Execute(); err != nil {
		return fmt.Errorf("failed to execute command %q: %w", name, err)
	}
	return nil
}
