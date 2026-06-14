---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how TikTok serves its data, not a
bug.

## A command exits 4 (walled)

This is the common one. TikTok puts its signed `/api/*` surface, and sometimes
its profile pages, behind a Web Application Firewall that scores the caller's IP,
headers, and session. From a residential browser session it answers. From a
datacenter IP, a VPN, or a cloud host it often serves a `Please wait...`
challenge or an empty body instead.

When `tt` sees that, it exits 4 with a clear message rather than pretending it
found nothing. The data is real and reachable, just not from where you are
calling. Run the command from a residential network, or stick to the SSR
commands (`video`, `hashtag`, `sound`, `raw`) which read the page's own JSON and
are not gated the same way.

The asymmetry is real and worth knowing: a video page usually serves its full
blob even from a datacenter IP, while the matching profile page may return the
challenge. So `tt video` tends to work where `tt user` and `tt posts` do not.

## Requests start failing or returning 429

TikTok rate-limits like any public site. `tt` already paces requests and retries
the transient failures, but a hard limit still means backing off. Raise the gap
between requests with `--delay` (for example `--delay 1s`) and retry later. A
burst of 429 or 5xx responses is the site asking you to slow down.

## Nothing is found for something you expected

The public surface is not the whole site. A private account returns a profile
shell with no items, and an age gated or removed video returns nothing. `tt`
exits 3 for a valid empty result, which is different from exit 4 for a gated one.
Check the input is spelled the way the site uses it, and try the same thing in a
private browser window before assuming it is missing.

## A bare video id has no url

A `/video/{id}` url carries the author handle, so the record's `url` is exact.
A bare numeric id does not, so pass `--author` to fill it in:

```bash
tt video 7106594312292453675 --author tiktok
```

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`tt`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).
