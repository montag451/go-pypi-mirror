package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/montag451/go-pypi-mirror/cmd"
)

func printAvailableCommands() {
	fmt.Println("Available commands:")
	for _, name := range cmd.Names() {
		fmt.Println(name)
	}
}

func main() {
	if len(os.Args) <= 1 {
		printAvailableCommands()
		os.Exit(1)
	}
	name := os.Args[1]
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error)
	go func() {
		defer close(errCh)
		if err := cmd.Execute(ctx, name, os.Args[2:]); err != nil {
			errCh <- fmt.Errorf("failed to execute command %q: %w", name, err)
		}
	}()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	select {
	case err, ok := <-errCh:
		if ok {
			log.Fatal(err)
		}
	case <-sigCh:
		cancel()
		if err, ok := <-errCh; ok {
			log.Fatal(err)
		}
	}
}
