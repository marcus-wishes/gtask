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
	Register(&RmCmd{})
}

// RmCmd implements the rm command.
type RmCmd struct {
	listName string
}

func (c *RmCmd) Name() string      { return "rm" }
func (c *RmCmd) Aliases() []string { return nil }
func (c *RmCmd) Synopsis() string  { return "Delete a task" }
func (c *RmCmd) Usage() string     { return "gtask rm [--list <list-name>] <ref>" }
func (c *RmCmd) NeedsAuth() bool   { return true }

func (c *RmCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
	fs.StringVar(&c.listName, "l", "", "")
}

func (c *RmCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Parse task reference
	ref, err := ParseTaskRef(args)
	if err != nil {
		if err == ErrTaskRefRequired {
			fmt.Fprintln(errOut, "error: task reference required")
		} else {
			fmt.Fprintf(errOut, "error: %v\n", err)
		}
		return exitcode.UserError
	}

	// Check mutual exclusivity: --list flag and list letter cannot both be used
	if c.listName != "" && ref.HasLetter {
		fmt.Fprintln(errOut, "error: cannot use both --list and list letter")
		return exitcode.UserError
	}

	// Validate task number
	if ref.TaskNum < 1 {
		fmt.Fprintf(errOut, "error: task number out of range: %d\n", ref.TaskNum)
		return exitcode.UserError
	}

	// Resolve list
	var list service.TaskList
	if c.listName != "" {
		// --list flag provided
		list, err = svc.ResolveList(ctx, c.listName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Fprintf(errOut, "error: list not found: %s\n", c.listName)
				return exitcode.UserError
			}
			if strings.Contains(err.Error(), "ambiguous") {
				fmt.Fprintf(errOut, "error: ambiguous list name: %s\n", c.listName)
				return exitcode.UserError
			}
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
	} else if ref.HasLetter {
		// List letter provided (e.g., a1, b 3)
		list, err = ResolveListByLetter(ctx, svc, ref.Letter)
		if err != nil {
			if strings.Contains(err.Error(), "list letter not found") {
				fmt.Fprintf(errOut, "error: list letter not found: %c\n", ref.Letter)
				return exitcode.UserError
			}
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
	} else {
		// Default list
		list, err = svc.DefaultList(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
	}

	// Find task by number
	task, err := findTaskByNumber(ctx, svc, list.ID, ref.TaskNum)
	if err != nil {
		if strings.Contains(err.Error(), "out of range") {
			fmt.Fprintf(errOut, "error: task number out of range: %d\n", ref.TaskNum)
			return exitcode.UserError
		}
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Delete task
	if err := svc.DeleteTask(ctx, list.ID, task.ID); err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
