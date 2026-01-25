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
	Register(&DoneCmd{})
}

// DoneCmd implements the done command.
type DoneCmd struct {
	listName string
}

func (c *DoneCmd) Name() string      { return "done" }
func (c *DoneCmd) Aliases() []string { return nil }
func (c *DoneCmd) Synopsis() string  { return "Mark a task completed" }
func (c *DoneCmd) Usage() string     { return "gtask done [--list <list-name>] <ref>" }
func (c *DoneCmd) NeedsAuth() bool   { return true }

func (c *DoneCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
}

func (c *DoneCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
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

	// Find task by number (fetch pages until we find it)
	task, err := findTaskByNumber(ctx, svc, list.ID, taskNum)
	if err != nil {
		if strings.Contains(err.Error(), "out of range") {
			fmt.Fprintf(errOut, "error: task number out of range: %d\n", taskNum)
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
