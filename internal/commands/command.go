// Package commands provides the command interface and implementations.
package commands

import (
	"context"
	"flag"
	"io"

	"gtask/internal/config"
	"gtask/internal/service"
)

// Command defines the interface for CLI commands.
type Command interface {
	// Name returns the primary command name.
	Name() string

	// Aliases returns alternative names for the command.
	Aliases() []string

	// Synopsis returns a short description for help output.
	Synopsis() string

	// Usage returns the usage string for help output.
	Usage() string

	// NeedsAuth returns true if the command requires authentication.
	// Commands like help, version, login, logout return false.
	NeedsAuth() bool

	// RegisterFlags registers command-specific flags.
	RegisterFlags(fs *flag.FlagSet)

	// Run executes the command.
	// cfg is always provided (config dir, paths).
	// svc is nil if NeedsAuth() returns false.
	// args contains positional arguments after flag parsing.
	// Returns exit code.
	Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int
}
