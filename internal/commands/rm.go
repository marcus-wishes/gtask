package commands

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
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
}

func (c *RmCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Check for task reference
	if len(args) == 0 {
		fmt.Fprintln(errOut, "error: task reference required")
		return exitcode.UserError
	}

	ref := args[0]

	// Parse task number (v1 only supports numeric references)
	taskNum, err := strconv.Atoi(ref)
	if err != nil {
		fmt.Fprintf(errOut, "error: invalid task reference: %s\n", ref)
		return exitcode.UserError
	}
	if taskNum < 1 {
		fmt.Fprintf(errOut, "error: task number out of range: %d\n", taskNum)
		return exitcode.UserError
	}

	// Resolve list
	var list service.TaskList
	if c.listName != "" {
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
	} else {
		list, err = svc.DefaultList(ctx)
		if err != nil {
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
	}

	// Find task by number
	task, err := findTaskByNumber(ctx, svc, list.ID, taskNum)
	if err != nil {
		if strings.Contains(err.Error(), "out of range") {
			fmt.Fprintf(errOut, "error: task number out of range: %d\n", taskNum)
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
