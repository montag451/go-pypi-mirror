package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
)

type command interface {
	FlagSet() *flag.FlagSet
	Execute(ctx context.Context) error
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
		for cmd := range commands {
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
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execChan := make(chan error)
	go func() {
		if err := cmd.Execute(cancelCtx); err != nil {
			execChan <- fmt.Errorf("failed to execute command %q: %w", name, err)
		} else {
			execChan <- nil
		}
		close(execChan)

	}()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	select {
	case err := <-execChan:
		if err != nil {
			fmt.Println(err)
		}
	case <-sigChan:
		cancel()
		err := <-execChan
		fmt.Println(err)
	}
	return nil
}
