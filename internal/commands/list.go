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
	"gtask/internal/output"
	"gtask/internal/service"
)

func init() {
	Register(&ListCmd{})
}

// ListCmd implements the list command.
// Handles both `gtask` (no args) and `gtask list <list-name>`.
type ListCmd struct {
	page int
}

// SetPage sets the page number (for testing).
func (c *ListCmd) SetPage(page int) {
	c.page = page
}

func (c *ListCmd) Name() string      { return "list" }
func (c *ListCmd) Aliases() []string { return nil }
func (c *ListCmd) Synopsis() string  { return "List tasks" }
func (c *ListCmd) Usage() string     { return "gtask list [--page <n>] <list-name>" }
func (c *ListCmd) NeedsAuth() bool   { return true }

func (c *ListCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.IntVar(&c.page, "page", 1, "")
}

func (c *ListCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Validate page number
	if c.page < 1 {
		fmt.Fprintf(errOut, "error: invalid page number: %d\n", c.page)
		return exitcode.UserError
	}

	// If no args, list all tasks (default + named lists)
	if len(args) == 0 {
		return c.listAll(ctx, cfg, svc, out, errOut)
	}

	// Otherwise, list specific list
	listName := strings.Join(args, " ")
	return c.listOne(ctx, cfg, svc, listName, out, errOut)
}

// listAll lists tasks from all lists (gtask with no args).
func (c *ListCmd) listAll(ctx context.Context, cfg *config.Config, svc service.Service, out, errOut io.Writer) int {
	hasAnyTasks := false

	// Get default list tasks (page 1 only for gtask with no args)
	defaultList, err := svc.DefaultList(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	defaultTasks, err := svc.ListOpenTasks(ctx, defaultList.ID, 1)
	if err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Print default list tasks (no header)
	for i, task := range defaultTasks {
		output.FormatTask(out, i+1, task)
		hasAnyTasks = true
	}

	// Get all lists
	lists, err := svc.ListLists(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Print named lists with tasks, assigning letters a-z
	letter := 'a'
	for _, list := range lists {
		if list.IsDefault {
			continue // Already printed
		}

		tasks, err := svc.ListOpenTasks(ctx, list.ID, 1)
		if err != nil {
			// Partial failure: print what we have so far, then error
			fmt.Fprintf(errOut, "error: failed to fetch list: %s: %v\n", list.Title, err)
			return exitcode.BackendError
		}

		if len(tasks) == 0 {
			continue // Skip empty lists
		}

		// Check for max 26 lists limit
		if letter > 'z' {
			fmt.Fprintln(errOut, "error: too many lists (max 26)")
			return exitcode.UserError
		}

		// Print list section with current letter
		output.FormatListHeader(out, list.Title, false)
		for i, task := range tasks {
			output.FormatTaskWithLetter(out, letter, i+1, task)
		}
		letter++
		hasAnyTasks = true
	}

	// If no tasks found anywhere
	if !hasAnyTasks && !cfg.Quiet {
		fmt.Fprintln(out, "no tasks found")
	}

	return exitcode.Success
}

// listOne lists tasks from a specific list (gtask list <name>).
func (c *ListCmd) listOne(ctx context.Context, cfg *config.Config, svc service.Service, listName string, out, errOut io.Writer) int {
	// Validate list name
	listName = strings.TrimSpace(listName)
	if listName == "" {
		fmt.Fprintln(errOut, "error: list name required")
		return exitcode.UserError
	}

	// Resolve list
	list, err := svc.ResolveList(ctx, listName)
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

	// Get tasks for the page
	tasks, err := svc.ListOpenTasks(ctx, list.ID, c.page)
	if err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	// Print list section (even if empty)
	output.FormatListHeader(out, list.Title, list.IsDefault)

	// Calculate starting number based on page
	startNum := (c.page-1)*100 + 1

	for i, task := range tasks {
		output.FormatTaskIndented(out, startNum+i, task)
	}

	return exitcode.Success
}

// parsePageFlag handles custom parsing for --page flag.
func parsePageFlag(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, fmt.Errorf("invalid page number: %s", s)
	}
	return n, nil
}
