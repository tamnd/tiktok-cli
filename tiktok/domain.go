package tiktok

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes TikTok as a kit Domain: a driver a multi-domain host (ant)
// enables with a single blank import,
//
//	import _ "github.com/tamnd/tiktok-cli/tiktok"
//
// the way a database/sql program enables a driver with `import _
// "github.com/lib/pq"`. The init below registers it; the host then dereferences
// tiktok:// URIs by routing to the operations Register installs. The standalone
// tt binary calls Register through cli.NewApp and shares the same registry, so
// the CLI and the host expose one set of operations.
func init() { kit.Register(Domain{}) }

// Domain is the TikTok driver. It carries no state; the per-run client is built
// by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against, and
// the identity a host reuses for help and version.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme:   "tiktok",
		Aliases:  []string{"tt"},
		Hosts:    []string{"tiktok.com", "www.tiktok.com", "vm.tiktok.com"},
		Identity: BaseIdentity(),
	}
}

// BaseIdentity is the help and version identity shared by the standalone binary
// and any host that links the package.
func BaseIdentity() kit.Identity {
	return kit.Identity{
		Binary: "tt",
		Short:  "A command line for TikTok.",
		Long: `tt reads public TikTok data and prints clean, pipeable records.

It needs no API key and no login. It reads the same public web surface a
logged-out browser sees: the server rendered universal-data blob on each page,
and the www.tiktok.com/api/* endpoints that the page's own JavaScript calls,
signed the way the web client signs them.

Records come out as table, JSON, JSONL, CSV, TSV, url, or raw.

tt is an independent tool and is not affiliated with ByteDance or TikTok.`,
		Site: Host,
		Repo: "https://github.com/tamnd/tiktok-cli",
	}
}

// Defaults seeds the framework baseline from the tiktok defaults, so an unset
// --rate/--retries/--timeout keeps the library's own pacing.
func Defaults(c *kit.Config) {
	d := DefaultConfig()
	c.Rate = d.Rate
	c.Timeout = d.Timeout
	c.Retries = d.Retries
	c.UserAgent = d.UserAgent
}

// Register installs the client factory and every TikTok operation onto app. It
// is the single point both surfaces go through: cli.NewApp calls it for the
// standalone binary, and a host calls Domain.Register for tiktok:// URIs.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)
	registerOps(app)
}

// Register is the convenience a host or the binary calls without naming the
// zero-value Domain.
func Register(app *kit.App) { Domain{}.Register(app) }

// Session is the per-run client kit injects into every operation. It pairs the
// HTTP client with the resolved quiet flag, so an operation can pace its own
// stderr progress without reaching for a global.
type Session struct {
	Client *Client
	Quiet  bool
}

// progressf prints a one-line progress note to stderr unless the run is quiet.
func (s *Session) Progressf(format string, args ...any) {
	if s == nil || s.Quiet {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// newClient is the factory kit calls once per run. It overlays the resolved
// framework globals on the library defaults so --rate, --timeout, --retries, and
// a custom User-Agent reach the HTTP client.
func newClient(_ context.Context, c kit.Config) (any, error) {
	cfg := DefaultConfig()
	if c.UserAgent != "" {
		cfg.UserAgent = c.UserAgent
	}
	if c.Rate > 0 {
		cfg.Rate = c.Rate
	}
	if c.Timeout > 0 {
		cfg.Timeout = c.Timeout
	}
	if c.Retries > 0 {
		cfg.Retries = c.Retries
	}
	return &Session{Client: NewClient(cfg), Quiet: c.Quiet}, nil
}

// Classify turns any accepted input into the canonical (type, id), so `ant
// resolve` and `ant url` need no network. A bare handle, a /video/ link, a
// /tag/ link, and a /music/ link each map to their resource type.
func (Domain) Classify(input string) (uriType, id string, err error) {
	if id, e := ParseVideoID(input); e == nil {
		return "video", id, nil
	}
	if id, e := ParseMusicID(input); e == nil {
		return "sound", id, nil
	}
	if name, ok := tagName(input); ok {
		return "hashtag", name, nil
	}
	if h, e := ParseHandle(input); e == nil {
		return "user", h, nil
	}
	return "", "", errs.Usage("unrecognized TikTok reference: %q", input)
}

// Locate is the inverse: the live https URL for a (type, id), built without a
// fetch.
func (Domain) Locate(uriType, id string) (string, error) {
	switch uriType {
	case "user":
		return Host + "/@" + id, nil
	case "video":
		return videoURL("", id), nil
	case "hashtag":
		return Host + "/tag/" + id, nil
	case "sound":
		return Host + "/music/x-" + id, nil
	default:
		return "", errs.Usage("tiktok has no resource type %q", uriType)
	}
}

// mapErr converts a library error into the kit error kind that carries the right
// exit code, so a host renders the same walled and not-found outcomes the
// standalone binary does. A WAF challenge is exit 4 (needs a residential
// session), a missing record is exit 6.
func MapErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ErrWalled):
		return errs.NeedAuth("%s", err.Error())
	case errors.Is(err, ErrNotFound):
		return errs.NotFound("%s", err.Error())
	default:
		return err
	}
}
