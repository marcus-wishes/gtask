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
	Register(&DoneCmd{})
}

// DoneCmd implements the done command.
type DoneCmd struct {
	listName string
}

// SetListName sets the list name (for testing).
func (c *DoneCmd) SetListName(name string) {
	c.listName = name
}

func (c *DoneCmd) Name() string      { return "done" }
func (c *DoneCmd) Aliases() []string { return nil }
func (c *DoneCmd) Synopsis() string  { return "Mark a task completed" }
func (c *DoneCmd) Usage() string     { return "gtask done [--list <list-name>] <ref>" }
func (c *DoneCmd) NeedsAuth() bool   { return true }

func (c *DoneCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
	fs.StringVar(&c.listName, "l", "", "")
}

func (c *DoneCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
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

	// Find task by number (fetch pages until we find it)
	task, err := findTaskByNumber(ctx, svc, list.ID, ref.TaskNum)
	if err != nil {
		if strings.Contains(err.Error(), "out of range") {
			fmt.Fprintf(errOut, "error: task number out of range: %d\n", ref.TaskNum)
			return exitcode.UserError
		}
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Complete task
	if err := svc.CompleteTask(ctx, list.ID, task.ID); err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}

// findTaskByNumber finds a task by its 1-based number in the list.
// Fetches pages as needed until the task is found.
func findTaskByNumber(ctx context.Context, svc service.Service, listID string, num int) (service.Task, error) {
	const pageSize = 100

	page := (num-1)/pageSize + 1
	indexInPage := (num - 1) % pageSize

	tasks, err := svc.ListOpenTasks(ctx, listID, page)
	if err != nil {
		return service.Task{}, err
	}

	if indexInPage >= len(tasks) {
		return service.Task{}, fmt.Errorf("task number out of range: %d", num)
	}

	return tasks[indexInPage], nil
}
