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
  gtask                                              List all open tasks
  gtask list [common flags] [--page <n>] <list-name> List tasks in a specific list
  gtask add [common flags] [--list <list-name>] <title...>
  gtask create [common flags] [--list <list-name>] <title...>
  gtask done [common flags] [--list <list-name>] <ref>
  gtask rm [common flags] [--list <list-name>] <ref>
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
`
