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
	Register(&AddCmd{})
	Register(&CreateCmd{})
}

// AddCmd implements the add command.
type AddCmd struct {
	listName string
}

// SetListName sets the list name (for testing).
func (c *AddCmd) SetListName(name string) {
	c.listName = name
}

func (c *AddCmd) Name() string      { return "add" }
func (c *AddCmd) Aliases() []string { return nil }
func (c *AddCmd) Synopsis() string  { return "Create a task" }
func (c *AddCmd) Usage() string     { return "gtask add [--list <list-name>] <title...>" }
func (c *AddCmd) NeedsAuth() bool   { return true }

func (c *AddCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
	fs.StringVar(&c.listName, "l", "", "")
}

func (c *AddCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	return runAdd(ctx, cfg, svc, c.listName, args, out, errOut)
}

// CreateCmd is an alias for AddCmd.
type CreateCmd struct {
	listName string
}

func (c *CreateCmd) Name() string      { return "create" }
func (c *CreateCmd) Aliases() []string { return nil }
func (c *CreateCmd) Synopsis() string  { return "Create a task (alias for add)" }
func (c *CreateCmd) Usage() string     { return "gtask create [--list <list-name>] <title...>" }
func (c *CreateCmd) NeedsAuth() bool   { return true }

func (c *CreateCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
	fs.StringVar(&c.listName, "l", "", "")
}

func (c *CreateCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	return runAdd(ctx, cfg, svc, c.listName, args, out, errOut)
}

// runAdd is the shared implementation for add and create commands.
func runAdd(ctx context.Context, cfg *config.Config, svc service.Service, listName string, args []string, out, errOut io.Writer) int {
	// Check for title
	if len(args) == 0 {
		fmt.Fprintln(errOut, "error: title required")
		return exitcode.UserError
	}

	// Join args to form title
	title := strings.Join(args, " ")
	if strings.TrimSpace(title) == "" {
		fmt.Fprintln(errOut, "error: title required")
		return exitcode.UserError
	}

	// Resolve list
	var list service.TaskList
	var err error
	if listName != "" {
		list, err = svc.ResolveList(ctx, listName)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				fmt.Fprintf(errOut, "error: list not found: %s\n", listName)
				return exitcode.UserError
			}
			if strings.Contains(err.Error(), "ambiguous") {
				fmt.Fprintf(errOut, "error: ambiguous list name: %s\n", listName)
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

	// Create task
	if err := svc.CreateTask(ctx, list.ID, title); err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
