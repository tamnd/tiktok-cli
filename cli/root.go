// Package cli assembles the tt command tree on top of the tiktok library and
// the any-cli/kit framework. The record-stream commands are kit operations the
// tiktok package declares once and exposes as CLI, HTTP, and MCP. The two
// commands that do not fit that shape, the universal-data byte dump and the
// version banner, are escape-hatch kit.Command commands that share the run state
// through the context.
package cli

import (
	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/tiktok-cli/tiktok"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// NewApp builds the kit application: the shared identity, the tiktok defaults,
// every record operation and the tiktok:// driver (both installed by
// tiktok.Register), and the escape-hatch commands.
func NewApp() *kit.App {
	id := tiktok.BaseIdentity()
	id.Version = Version

	app := kit.New(id, kit.WithDefaults(tiktok.Defaults))
	tiktok.Register(app)

	// kit gives every binary an honest default User-Agent. The standalone tt
	// kept a --user-agent override, so restore it here: bind it to a local and
	// fold it onto the resolved config the client factory reads.
	var userAgent string
	app.GlobalFlags(func(f *kit.FlagSet) {
		f.StringVar(&userAgent, "user-agent", "", "override the User-Agent sent with each request")
	})
	app.Finalize(func(c *kit.Config) {
		if userAgent != "" {
			c.UserAgent = userAgent
		}
	})

	app.AddCommand(newRawCmd())
	app.AddCommand(newVersionCmd())
	return app
}
