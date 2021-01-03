package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"sort"
)

type Command interface {
	FlagSet() *flag.FlagSet
	Execute(ctx context.Context) error
}

var ErrCommandNotFound = errors.New("command not found")

var commands = map[string]Command{}

func RegisterCommand(cmd Command) {
	name := cmd.FlagSet().Name()
	if _, ok := commands[name]; ok {
		panic(fmt.Sprintf("command %q already registered", name))
	}
	commands[name] = cmd
}

func Names() []string {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func Execute(ctx context.Context, cmdName string, args []string) error {
	cmd, ok := commands[cmdName]
	if !ok {
		return ErrCommandNotFound
	}
	cmd.FlagSet().Parse(args)
	return cmd.Execute(ctx)
}
