package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/service"
)

func init() {
	Register(&CreateListCmd{})
	Register(&AddListCmd{})
}

// CreateListCmd implements the createlist command.
type CreateListCmd struct{}

func (c *CreateListCmd) Name() string      { return "createlist" }
func (c *CreateListCmd) Aliases() []string { return nil }
func (c *CreateListCmd) Synopsis() string  { return "Create a new list" }
func (c *CreateListCmd) Usage() string     { return "gtask createlist [common flags] <list-name>" }
func (c *CreateListCmd) NeedsAuth() bool   { return true }

func (c *CreateListCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *CreateListCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	return runCreateList(ctx, cfg, svc, args, out, errOut)
}

// AddListCmd is an alias for CreateListCmd.
type AddListCmd struct{}

func (c *AddListCmd) Name() string      { return "addlist" }
func (c *AddListCmd) Aliases() []string { return nil }
func (c *AddListCmd) Synopsis() string  { return "Create a new list (alias for createlist)" }
func (c *AddListCmd) Usage() string     { return "gtask addlist [common flags] <list-name>" }
func (c *AddListCmd) NeedsAuth() bool   { return true }

func (c *AddListCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *AddListCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	return runCreateList(ctx, cfg, svc, args, out, errOut)
}

// runCreateList is the shared implementation for createlist and addlist commands.
func runCreateList(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Check for list name
	if len(args) == 0 {
		fmt.Fprintln(errOut, "error: list name required")
		return exitcode.UserError
	}

	// Join args to form list name
	name := strings.Join(args, " ")
	name = strings.TrimSpace(name)
	if name == "" {
		fmt.Fprintln(errOut, "error: list name required")
		return exitcode.UserError
	}

	// Check if list already exists
	_, err := svc.ResolveList(ctx, name)
	if err == nil {
		// List found - already exists
		fmt.Fprintf(errOut, "error: list already exists: %s\n", name)
		return exitcode.UserError
	}
	// If error is not "not found", it's a backend error
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "ambiguous") {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Create list
	if err := svc.CreateList(ctx, name); err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
