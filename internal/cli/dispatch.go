package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"gtask/internal/commands"
	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/service"
)

// ServiceFactory creates a Service from config.
// Used to inject the backend during dispatch.
type ServiceFactory func(ctx context.Context, cfg *config.Config) (service.Service, error)

// Dispatcher handles command-line parsing and dispatch.
type Dispatcher struct {
	registry *commands.Registry
	factory  ServiceFactory
}

// NewDispatcher creates a new dispatcher with the given registry and service factory.
func NewDispatcher(registry *commands.Registry, factory ServiceFactory) *Dispatcher {
	return &Dispatcher{
		registry: registry,
		factory:  factory,
	}
}

// Run parses arguments and dispatches to the appropriate command.
// Returns the exit code.
func (d *Dispatcher) Run(ctx context.Context, args []string, out, errOut io.Writer) int {
	// No args -> dispatch to "list" command with no args
	if len(args) == 0 {
		return d.dispatch(ctx, "list", nil, out, errOut)
	}

	cmdName := args[0]

	// If first token starts with -, it's an error (flags require a command)
	if strings.HasPrefix(cmdName, "-") {
		fmt.Fprintf(errOut, "error: unknown command: %s\n", cmdName)
		return exitcode.UserError
	}

	// Look up command
	cmd, ok := d.registry.Find(cmdName)
	if !ok {
		fmt.Fprintf(errOut, "error: unknown command: %s\n", cmdName)
		return exitcode.UserError
	}

	// Parse flags
	remaining := args[1:]
	return d.dispatchCommand(ctx, cmd, remaining, out, errOut)
}

func (d *Dispatcher) dispatch(ctx context.Context, cmdName string, args []string, out, errOut io.Writer) int {
	cmd, ok := d.registry.Find(cmdName)
	if !ok {
		fmt.Fprintf(errOut, "error: unknown command: %s\n", cmdName)
		return exitcode.UserError
	}
	return d.dispatchCommand(ctx, cmd, args, out, errOut)
}

func (d *Dispatcher) dispatchCommand(ctx context.Context, cmd commands.Command, args []string, out, errOut io.Writer) int {
	// Create flag set with custom error handling
	fs := flag.NewFlagSet(cmd.Name(), flag.ContinueOnError)
	fs.SetOutput(io.Discard) // We handle errors ourselves

	// Common flags
	var configDir string
	var quiet bool
	var debug bool

	fs.StringVar(&configDir, "config", "", "")
	fs.BoolVar(&quiet, "quiet", false, "")
	fs.BoolVar(&debug, "debug", false, "")

	// Register command-specific flags
	cmd.RegisterFlags(fs)

	// Parse flags
	if err := fs.Parse(args); err != nil {
		// Handle specific error types
		errStr := err.Error()

		// Check for missing flag value
		if strings.Contains(errStr, "needs a value") || strings.Contains(errStr, "flag needs an argument") {
			// Extract flag name
			parts := strings.Split(errStr, ":")
			if len(parts) > 0 {
				flagPart := strings.TrimSpace(parts[0])
				flagPart = strings.TrimPrefix(flagPart, "flag ")
				fmt.Fprintf(errOut, "error: flag needs an argument: %s\n", flagPart)
				return exitcode.UserError
			}
		}

		// Check for unknown flag
		if strings.HasPrefix(errStr, "flag provided but not defined:") {
			flagName := strings.TrimPrefix(errStr, "flag provided but not defined: ")
			fmt.Fprintf(errOut, "error: unknown flag: %s\n", flagName)
			return exitcode.UserError
		}

		// Generic error handling for bad flag values
		if strings.Contains(errStr, "invalid value") {
			fmt.Fprintf(errOut, "error: %s\n", errStr)
			return exitcode.UserError
		}

		fmt.Fprintf(errOut, "error: %s\n", errStr)
		return exitcode.UserError
	}

	// Check if first positional arg starts with - (should have been parsed as flag)
	positionalArgs := fs.Args()
	if len(positionalArgs) > 0 && strings.HasPrefix(positionalArgs[0], "-") {
		fmt.Fprintf(errOut, "error: unknown flag: %s\n", positionalArgs[0])
		return exitcode.UserError
	}

	// Create config
	cfg, err := config.New(configDir)
	if err != nil {
		fmt.Fprintf(errOut, "error: %s\n", err)
		return exitcode.UserError
	}
	cfg.Quiet = quiet
	cfg.Debug = debug

	// Check auth requirements
	var svc service.Service
	if cmd.NeedsAuth() {
		if d.factory != nil {
			// Custom factory provided (e.g., tests with FakeService) - skip file checks,
			// let the factory handle auth
			svc, err = d.factory(ctx, cfg)
			if err != nil {
				// Check if it's an auth error
				if strings.Contains(err.Error(), "token") || strings.Contains(err.Error(), "auth") {
					fmt.Fprintf(errOut, "error: auth error: %s\n", err)
					return exitcode.AuthError
				}
				fmt.Fprintf(errOut, "error: backend error: %s\n", err)
				return exitcode.BackendError
			}
		} else {
			// No factory - check for required auth files and report user-friendly errors
			if !cfg.HasOAuthClient() {
				fmt.Fprintf(errOut, "error: oauth_client.json not found in %s\n", cfg.Dir)
				return exitcode.AuthError
			}
			if !cfg.HasToken() {
				fmt.Fprintf(errOut, "error: not logged in (run: gtask login)\n")
				return exitcode.AuthError
			}
			// No factory and no service creation - svc remains nil
			// Commands must handle nil service (this path is for pre-flight checks only)
		}
	}

	// Run command
	return cmd.Run(ctx, cfg, svc, positionalArgs, out, errOut)
}
