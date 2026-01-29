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
func (c *DoneCmd) Usage() string     { return "gtask done [--list <list-name>] <ref>..." }
func (c *DoneCmd) NeedsAuth() bool   { return true }

func (c *DoneCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.listName, "list", "", "")
	fs.StringVar(&c.listName, "l", "", "")
}

func (c *DoneCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Parse task references
	refs, err := ParseTaskRefs(args)
	if err != nil {
		if err == ErrTaskRefRequired {
			fmt.Fprintln(errOut, "error: task reference required")
		} else {
			fmt.Fprintf(errOut, "error: %v\n", err)
		}
		return exitcode.UserError
	}

	hasLetter := false
	for _, ref := range refs {
		if ref.HasLetter {
			hasLetter = true
			break
		}
	}

	// Check mutual exclusivity: --list flag and list letters cannot both be used.
	if c.listName != "" && hasLetter {
		fmt.Fprintln(errOut, "error: cannot use both --list and list letter")
		return exitcode.UserError
	}

	// Validate task numbers before any backend calls.
	for _, ref := range refs {
		if ref.TaskNum < 1 {
			fmt.Fprintf(errOut, "error: task number out of range: %d\n", ref.TaskNum)
			return exitcode.UserError
		}
	}

	// Resolve list context(s).
	var defaultList service.TaskList
	var listByLetter map[rune]service.TaskList

	if c.listName == "" {
		needsDefault := false
		for _, ref := range refs {
			if !ref.HasLetter {
				needsDefault = true
				break
			}
		}
		if needsDefault {
			var err error
			defaultList, err = svc.DefaultList(ctx)
			if err != nil {
				fmt.Fprintf(errOut, "error: backend error: %v\n", err)
				return exitcode.BackendError
			}
		}

		if hasLetter {
			var err error
			listByLetter, err = BuildListLetterMap(ctx, svc)
			if err != nil {
				if err == ErrTooManyLists {
					fmt.Fprintln(errOut, "error: too many lists (max 26)")
					return exitcode.UserError
				}
				fmt.Fprintf(errOut, "error: backend error: %v\n", err)
				return exitcode.BackendError
			}
		}
	}

	var listFromFlag service.TaskList
	if c.listName != "" {
		var err error
		listFromFlag, err = svc.ResolveList(ctx, c.listName)
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
	}

	// Resolve all task refs (listID + taskID) first, then mutate.
	type target struct {
		listID string
		taskID string
	}
	var targets []target
	seen := make(map[string]struct{})
	cache := make(taskPageCache)

	for _, ref := range refs {
		var listID string
		if c.listName != "" {
			listID = listFromFlag.ID
		} else if ref.HasLetter {
			list, ok := listByLetter[ref.Letter]
			if !ok {
				fmt.Fprintf(errOut, "error: list letter not found: %c\n", ref.Letter)
				return exitcode.UserError
			}
			listID = list.ID
		} else {
			listID = defaultList.ID
		}

		task, err := findTaskByNumberCached(ctx, svc, listID, ref.TaskNum, cache)
		if err != nil {
			if strings.Contains(err.Error(), "out of range") {
				fmt.Fprintf(errOut, "error: task number out of range: %d\n", ref.TaskNum)
				return exitcode.UserError
			}
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}

		key := listID + "\x00" + task.ID
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, target{listID: listID, taskID: task.ID})
	}

	for _, t := range targets {
		if err := svc.CompleteTask(ctx, t.listID, t.taskID); err != nil {
			fmt.Fprintf(errOut, "error: backend error: %v\n", err)
			return exitcode.BackendError
		}
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
