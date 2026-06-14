// Package cli builds the tt command tree on top of the tiktok library.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tamnd/tiktok-cli/tiktok"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// App holds shared state threaded through every command.
type App struct {
	client *tiktok.Client
	cfg    tiktok.Config

	output   string
	fields   []string
	noHeader bool
	template string
	limit    int
	jobs     int
	quiet    bool
}

// Root builds the root command and its subtree.
func Root() *cobra.Command {
	app := &App{cfg: tiktok.DefaultConfig()}

	root := &cobra.Command{
		Use:   "tt",
		Short: "A command line for TikTok.",
		Long: `tt reads public TikTok data and prints clean, pipeable records.

It needs no API key and no login. It reads the same public web surface a
logged-out browser sees: the server rendered universal-data blob on each page,
and the www.tiktok.com/api/* endpoints that the page's own JavaScript calls,
signed the way the web client signs them.

Records come out as table, JSON, JSONL, CSV, TSV, url, or raw.

tt is an independent tool and is not affiliated with ByteDance or TikTok.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return app.setup()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&app.output, "output", "o", "auto", "output: table|json|jsonl|csv|tsv|url|raw (auto=table on TTY, jsonl piped)")
	pf.StringSliceVar(&app.fields, "fields", nil, "comma-separated columns to include")
	pf.BoolVar(&app.noHeader, "no-header", false, "omit the header row in table/csv/tsv")
	pf.StringVar(&app.template, "template", "", "Go text/template applied per record")
	pf.IntVarP(&app.limit, "limit", "n", 0, "limit number of records (0 = command default)")
	pf.IntVarP(&app.jobs, "jobs", "j", 4, "concurrent fetches where a command pages or fans out")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress progress on stderr")

	pf.DurationVar(&app.cfg.Rate, "delay", app.cfg.Rate, "minimum spacing between requests")
	pf.DurationVar(&app.cfg.Timeout, "timeout", app.cfg.Timeout, "per-request timeout")
	pf.IntVar(&app.cfg.Retries, "retries", app.cfg.Retries, "retry attempts on 429/5xx")
	pf.StringVar(&app.cfg.UserAgent, "user-agent", app.cfg.UserAgent, "User-Agent sent with each request")

	root.AddCommand(
		app.userCmd(),
		app.videoCmd(),
		app.postsCmd(),
		app.commentsCmd(),
		app.repliesCmd(),
		app.searchCmd(),
		app.usersCmd(),
		app.hashtagCmd(),
		app.soundCmd(),
		app.trendingCmd(),
		app.discoverCmd(),
		app.rawCmd(),
		newVersionCmd(),
	)
	return root
}

func (a *App) setup() error {
	if a.output == "" || a.output == "auto" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			a.output = string(FormatTable)
		} else {
			a.output = string(FormatJSONL)
		}
	}
	if !Format(a.output).Valid() {
		return codeError(exitUsage, fmt.Errorf("unknown output format %q", a.output))
	}
	a.client = tiktok.NewClient(a.cfg)
	return nil
}

func (a *App) render(records any) error {
	r := NewRenderer(os.Stdout, Format(a.output), a.fields, a.noHeader, a.template)
	return r.Render(records)
}

// renderList renders records and maps an empty result to exit 3.
func (a *App) renderList(records any, n int) error {
	if err := a.render(records); err != nil {
		return err
	}
	if n == 0 {
		return codeError(exitNoData, nil)
	}
	return nil
}

func (a *App) progressf(format string, args ...any) {
	if a.quiet {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (a *App) effectiveLimit(def int) int {
	if a.limit > 0 {
		return a.limit
	}
	return def
}

// mapErr turns a library error into an ExitError with the right code. A WAF
// challenge maps to exit 4, a missing record to exit 3.
func mapErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, tiktok.ErrWalled):
		return codeError(exitWalled, err)
	case errors.Is(err, tiktok.ErrNotFound):
		return codeError(exitNoData, err)
	default:
		return codeError(exitError, err)
	}
}
