---
title: "Troubleshooting"
description: "The handful of things that trip people up, and how to fix each one."
weight: 40
---

Most of these come down to network reality or how tiktok serves its data,
not a bug. Fill this page out with the site-specific cases as you find them.

## Requests start failing or returning 429

tiktok rate-limits like any public site. tt already paces
requests and retries the transient failures, but a hard limit still means
backing off. Raise the delay between requests with `--rate` (for example
`--rate 1s`), lower any concurrency you have set, and retry later. A burst of
429 or 5xx responses is the site asking you to slow down, not a defect.

## Nothing is found for something you expected

The public surface is not the whole site. Some data sits behind a login, a
region, or a page that only renders with JavaScript, and that part is not
reachable without the right session. Check that the input is spelled the way the
site uses it, try a broader query, and see whether the same thing is visible in
a private browser window before assuming it is missing.

## A command needs a session

Where a surface is gated, tt reads a cookie or token you supply
rather than logging in for you. Pass it on the command that needs it and keep it
out of your shell history. Commands that work without one stay anonymous.

## The binary is not on your PATH

`go install` puts the binary in `$(go env GOPATH)/bin` (usually `~/go/bin`), and
a release archive leaves it wherever you unpacked it. If your shell cannot find
`tt`, add that directory to your `PATH`. See
[installation](/getting-started/installation/).

## Seeing what tt actually did

When something behaves unexpectedly, `-v` adds per-request detail so you can see
the URLs it hit and the responses it got. That is usually enough to tell a rate
limit apart from a genuinely empty result.
