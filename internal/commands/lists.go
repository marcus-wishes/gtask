package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/output"
	"gtask/internal/service"
)

func init() {
	Register(&ListsCmd{})
}

// ListsCmd implements the lists command.
type ListsCmd struct{}

func (c *ListsCmd) Name() string      { return "lists" }
func (c *ListsCmd) Aliases() []string { return nil }
func (c *ListsCmd) Synopsis() string  { return "Print all lists" }
func (c *ListsCmd) Usage() string     { return "gtask lists [common flags]" }
func (c *ListsCmd) NeedsAuth() bool   { return true }

func (c *ListsCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *ListsCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	lists, err := svc.ListLists(ctx)
	if err != nil {
		fmt.Fprintf(errOut, "error: backend error: %v\n", err)
		return exitcode.BackendError
	}

	for _, list := range lists {
		output.FormatListName(out, list)
	}

	return exitcode.Success
}
