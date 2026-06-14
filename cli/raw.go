package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/tiktok-cli/tiktok"
)

// newRawCmd prints a page's universal-data blob as pretty JSON. It does not fit
// the emit-records shape (the output is one opaque document, not a stream of
// typed records), so it is an escape-hatch command that pulls the run's client
// from the context the same way an operation receives it by injection.
func newRawCmd() kit.Command {
	return kit.Command{
		Use:   "raw <url>",
		Short: "Print a page's universal-data blob as pretty JSON",
		Group: "read",
		Args:  kit.ExactArgs(1),
		Run: func(ctx context.Context, args []string) error {
			sess := kit.MustClient[*tiktok.Session](ctx)
			sess.Progressf("fetching %s", args[0])
			out, err := sess.Client.RawUniversal(ctx, args[0])
			if err != nil {
				return tiktok.MapErr(err)
			}
			_, _ = fmt.Fprintln(os.Stdout, string(out))
			return nil
		},
	}
}
