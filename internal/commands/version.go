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

// Version is the application version. Set at build time.
var Version = "0.1.0"

func init() {
	Register(&VersionCmd{})
}

// VersionCmd implements the version command.
type VersionCmd struct{}

func (c *VersionCmd) Name() string      { return "version" }
func (c *VersionCmd) Aliases() []string { return nil }
func (c *VersionCmd) Synopsis() string  { return "Print version" }
func (c *VersionCmd) Usage() string     { return "gtask version" }
func (c *VersionCmd) NeedsAuth() bool   { return false }

func (c *VersionCmd) RegisterFlags(fs *flag.FlagSet) {}

func (c *VersionCmd) Run(ctx context.Context, cfg *config.Config, svc service.Service, args []string, out, errOut io.Writer) int {
	fmt.Fprintf(out, "gtask %s\n", Version)
	return exitcode.Success
}
