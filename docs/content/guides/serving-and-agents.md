---
title: "Serving and agents"
description: "Run the same commands over HTTP, as an MCP tool set, or as a tiktok:// resource driver."
weight: 80
---

Every read command is declared once and exposed four ways: as the CLI, as an
HTTP endpoint, as an MCP tool, and as a `tiktok://` resource driver. The data and
the records are identical across all four; only the front door changes. This
comes from the [any-cli/kit](https://github.com/tamnd/any-cli) framework.

## Over HTTP

`tt serve` puts the commands behind a small HTTP server that answers in NDJSON,
one record per line:

```bash
tt serve --addr :7777
```

Each command is a route under `/v1`, the argument is the next path segment, and
flags are query parameters:

```bash
curl 'http://127.0.0.1:7777/v1/video/7106594312292453675?author=tiktok'
curl 'http://127.0.0.1:7777/v1/user/tiktok'
curl 'http://127.0.0.1:7777/v1/hashtag/minecraft?videos=true&limit=50'
```

`/healthz` answers 200 for a liveness check. The plane rules still hold: a walled
surface comes back as HTTP 401 rather than a record stream, the server's analog
of the CLI's exit 4.

## As an MCP tool set

`tt mcp` speaks the Model Context Protocol over stdio, so an agent can call
`video`, `user`, `posts`, `search`, and the rest as tools:

```bash
tt mcp
```

Point an MCP-aware client at that command. It lists one tool per read command,
with the same arguments and flags the CLI takes, and returns the same records.

## As a tiktok:// resource driver

The package registers a `tiktok` domain the way a program registers a database
driver with `database/sql`. A host enables it with one blank import:

```go
import _ "github.com/tamnd/tiktok-cli/tiktok"
```

Then [ant](https://github.com/tamnd/ant), or any program that links the package,
dereferences `tiktok://` URIs:

```bash
ant get tiktok://user/tiktok                  # the profile record
ant get tiktok://video/7106594312292453675    # one video
ant ls  tiktok://user/tiktok                   # the user's videos
ant cat tiktok://video/7106594312292453675     # just the description text
ant url tiktok://hashtag/minecraft             # the live https URL
```

`get` reads a single record, `ls` lists the records under a resource, `cat`
prints the body text (a caption or a bio), and `url` resolves the live link.
