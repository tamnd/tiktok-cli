package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/tamnd/any-cli/kit"
)

// versionCmd holds the --short flag for the version command. kit gives the root
// a --version flag from the identity; this adds the familiar `tt version`
// subcommand with the full build line.
type versionCmd struct {
	short bool
}

func newVersionCmd() kit.Command {
	v := &versionCmd{}
	return kit.Command{
		Use:   "version",
		Short: "Print version information",
		Flags: v.flags,
		Run:   v.run,
	}
}

func (v *versionCmd) flags(f *kit.FlagSet) {
	f.BoolVar(&v.short, "short", false, "print just the version number")
}

func (v *versionCmd) run(_ context.Context, _ []string) error {
	if v.short {
		_, _ = fmt.Fprintln(os.Stdout, Version)
		return nil
	}
	_, _ = fmt.Fprintf(os.Stdout, "tt %s (commit %s, built %s, %s/%s, %s)\n",
		Version, Commit, Date, runtime.GOOS, runtime.GOARCH, runtime.Version())
	return nil
}
