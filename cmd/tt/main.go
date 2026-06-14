// Command tt is a single-binary command line for TikTok.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/tiktok-cli/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Build metadata lives on the cli package vars; goreleaser injects them with
	// -ldflags at release time. kit.Run drives the whole CLI and returns the
	// process exit code.
	os.Exit(kit.Run(ctx, cli.NewApp()))
}
