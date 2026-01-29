package commands

import (
	"context"
	"flag"
	"fmt"
	"io"

	"gtask/internal/config"
	"gtask/internal/exitcode"
	"gtask/internal/service"
)

func init() {
	Register(&HelpCmd{})
}

// HelpCmd implements the help command.
type HelpCmd struct{}

func (c *HelpCmd) Name() string      { return "help" }
func (c *HelpCmd) Aliases() []string { return nil }
func (c *HelpCmd) Synopsis() string  { return "Print usage" }
func (c *HelpCmd) Usage() string     { return "gtask help" }
func (c *HelpCmd) NeedsAuth() bool   { return false }

func (c *HelpCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *HelpCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	fmt.Fprint(out, helpText)
	return exitcode.Success
}

const helpText = `Usage:
  gtask                                              List all open tasks (with list letters)
  gtask list [common flags] [--page <n>] <list-name> List tasks in a specific list
  gtask add [common flags] [-l|--list <list-name>] <title...>
  gtask create [common flags] [-l|--list <list-name>] <title...>
  gtask done [common flags] [-l|--list <list-name>] <ref>...
  gtask done <number>                                Mark task done in the default list
  gtask done <letter><number>                        Mark task done using list letter (e.g., a1, b3)
  gtask rm [common flags] [-l|--list <list-name>] <ref>...
  gtask rm <number>                                  Delete task in the default list
  gtask rm <letter><number>                          Delete task using list letter
  gtask lists [common flags]
  gtask createlist [common flags] <list-name>
  gtask addlist [common flags] <list-name>
  gtask rmlist [common flags] [--force] <list-name>
  gtask login [common flags]
  gtask logout [common flags]
  gtask help
  gtask version

Common flags:
  --config <dir>   Override config directory
  --quiet          Suppress informational output
  --debug          Print debug logs to stderr

List letters (a-z) are shown in 'gtask' output and can be used with 'done' and 'rm'.
`
