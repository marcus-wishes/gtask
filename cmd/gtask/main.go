// Package main is the entry point for the gtask CLI.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gtask/internal/backend/googletasks"
	"gtask/internal/cli"
	"gtask/internal/commands"
	"gtask/internal/config"
	"gtask/internal/service"

	// Import all command packages to register them via init()
	_ "gtask/internal/commands"
)

func main() {
	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Create service factory
	factory := func(ctx context.Context, cfg *config.Config) (service.Service, error) {
		return googletasks.New(ctx, cfg)
	}

	// Create dispatcher
	dispatcher := cli.NewDispatcher(commands.DefaultRegistry, factory)

	// Run and exit with code
	code := dispatcher.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(code)
}
