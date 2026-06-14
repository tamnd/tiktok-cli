---
title: "Release notes"
linkTitle: "Release notes"
description: "What changed in each tt release, newest first."
weight: 40
---

What shipped in each release, newest first. Every tagged version builds the same
set of artifacts: archives for Linux, macOS, Windows, and FreeBSD, Linux
packages (deb, rpm, apk), a multi-arch container image on GHCR, and entries for
the package managers. Binaries are pure Go, so there is nothing to install
alongside them.

- [v0.2.1](/release-notes/v0-2-1/) — colored output on a terminal: bordered
  tables with colored headers and colorized JSON, plain when piped.
- [v0.2.0](/release-notes/v0-2-0/) — the commands now run on the any-cli/kit
  framework: the same reads serve over HTTP and MCP and back a `tiktok://`
  resource driver, with `--rate` and `--db` joining the global flags.
- [v0.1.0](/release-notes/v0-1-0/) — the first public release: the full read
  surface across both planes, the discovery walk, and the signing packages.
