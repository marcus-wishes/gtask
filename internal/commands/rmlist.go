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
	Register(&RmListCmd{})
}

// RmListCmd implements the rmlist command.
type RmListCmd struct {
	force bool
}

// SetForce sets the force flag (for testing).
func (c *RmListCmd) SetForce(force bool) {
	c.force = force
}

func (c *RmListCmd) Name() string      { return "rmlist" }
func (c *RmListCmd) Aliases() []string { return nil }
func (c *RmListCmd) Synopsis() string  { return "Delete a list" }
func (c *RmListCmd) Usage() string     { return "gtask rmlist [--force] <list-name>" }
func (c *RmListCmd) NeedsAuth() bool   { return true }

func (c *RmListCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.force, "force", false, "")
}

func (c *RmListCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
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

	// Resolve list
	list, err := svc.ResolveList(ctx, name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Fprintf(errOut, "error: list not found: %s\n", name)
			return exitcode.UserError
		}
		if strings.Contains(err.Error(), "ambiguous") {
			fmt.Fprintf(errOut, "error: ambiguous list name: %s\n", name)
			return exitcode.UserError
		}
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Cannot delete default list
	if list.IsDefault {
		fmt.Fprintln(errOut, "error: cannot delete default list")
		return exitcode.UserError
	}

	// Check if list is empty (unless --force)
	if !c.force {
		hasOpenTasks, err := svc.HasOpenTasks(ctx, list.ID)
		if err != nil {
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
		if hasOpenTasks {
			fmt.Fprintln(errOut, "error: list not empty (use --force)")
			return exitcode.UserError
		}
	}

	// Delete list
	if err := svc.DeleteList(ctx, list.ID); err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
