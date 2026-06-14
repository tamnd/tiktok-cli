---
title: "CLI"
description: "Every command and the flags that matter."
weight: 10
---

```
tt <command> [flags]
```

Run `tt <command> --help` for the full flag list on any command. Each command
notes which plane it rides: SSR commands read the page's own JSON and answer
reliably, API commands call the signed web endpoints and may be gated from a
datacenter IP.

## Commands

| Command | Plane | What it does |
|---|---|---|
| `user <handle>` | SSR | Profile record for a @handle |
| `video <url\|id>` | SSR | One video record with author, sound, hashtags, counters |
| `posts <handle\|secUid>` | API | A user's public videos, paged |
| `comments <url\|id>` | API | Comments under a video |
| `replies <url\|id> <comment-id>` | API | Replies under one comment |
| `search <query>` | API | Mixed search hits (videos and users) |
| `users <query>` | API | User search hits |
| `hashtag <name>` | SSR/API | Hashtag record, or its videos with `--videos` |
| `sound <url\|id>` | SSR/API | Sound record, or its videos with `--videos` |
| `trending` | API | Logged-out recommend feed |
| `raw <url>` | SSR | The page's universal-data blob as pretty JSON |
| `version` | | Print the version and exit |

## Arguments

- A `<handle>` accepts `tiktok`, `@tiktok`, or a full profile url.
- A video `<url|id>` accepts a `/video/{id}` url or a bare numeric id. Pass
  `--author` with a bare id to build the canonical url in the record.
- `posts` accepts a handle (it resolves the secUid from the profile first) or a
  secUid directly.

## Command flags

- `posts`: `--cursor` resumes from a paging cursor.
- `comments`: `--replies` expands every thread inline, `--author` sets the url
  field.
- `hashtag`, `sound`: `--videos` emits the video list instead of the header
  record.

## Global flags

| Flag | Default | Meaning |
|---|---|---|
| `-o, --output` | auto | `table\|json\|jsonl\|csv\|tsv\|url\|raw` |
| `--fields` | | comma-separated columns to include |
| `--no-header` | false | omit the header row in table/csv/tsv |
| `--template` | | Go text/template applied per record |
| `-n, --limit` | 0 | max records (0 = command default) |
| `-j, --jobs` | 4 | concurrent fetches where a command pages |
| `-q, --quiet` | false | suppress progress on stderr |
| `--delay` | 600ms | minimum spacing between requests |
| `--timeout` | 30s | per-request timeout |
| `--retries` | 5 | retry attempts on 429/5xx |
| `--user-agent` | desktop Chrome | User-Agent sent with each request |

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success, at least one record |
| 1 | error |
| 2 | usage error |
| 3 | no data (a valid empty result) |
| 4 | walled (the firewall gated this surface) |
