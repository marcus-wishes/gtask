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
	Register(&LogoutCmd{})
}

// LogoutCmd implements the logout command.
type LogoutCmd struct{}

func (c *LogoutCmd) Name() string      { return "logout" }
func (c *LogoutCmd) Aliases() []string { return nil }
func (c *LogoutCmd) Synopsis() string  { return "Remove stored credentials" }
func (c *LogoutCmd) Usage() string     { return "gtask logout [common flags]" }
func (c *LogoutCmd) NeedsAuth() bool   { return false }

func (c *LogoutCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *LogoutCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	// Check if token.json exists
	if !cfg.HasToken() {
		if !cfg.Quiet {
			fmt.Fprintln(out, "not logged in")
		}
		return exitcode.Success
	}

	// Delete token.json
	if err := cfg.RemoveToken(); err != nil {
		fmt.Fprintf(errOut, "error: failed to remove token: %v\n", err)
		return exitcode.AuthError
	}

	if !cfg.Quiet {
		fmt.Fprintln(out, "ok")
	}
	return exitcode.Success
}
