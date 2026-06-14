---
title: "The raw page blob"
description: "Print a page's whole universal-data JSON, the source the SSR commands read from."
weight: 70
---

The reliable commands (`video`, `user`, `hashtag`, `sound`) all read one thing:
the `__UNIVERSAL_DATA_FOR_REHYDRATION__` JSON blob a TikTok page ships inside its
HTML. `raw` prints that whole blob as pretty JSON, before `tt` picks it apart
into a record:

```bash
tt raw https://www.tiktok.com/@tiktok/video/7106594312292453675
```

Because it reads the page directly with no signing, `raw` answers from anywhere,
the same as `video`. It takes a url, not a bare id, since it fetches an actual
page.

## Why reach for it

`raw` is the escape hatch for the fields the typed records leave out. A record is
a flat, stable subset; the blob has everything the page loaded. Pull a value the
record does not carry with `jq`:

```bash
tt raw https://www.tiktok.com/@tiktok | jq '.__DEFAULT_SCOPE__ | keys'
```

From there you can drill into whatever node you need:

```bash
tt raw https://www.tiktok.com/@tiktok/video/7106594312292453675 \
  | jq '.__DEFAULT_SCOPE__["webapp.video-detail"].itemInfo.itemStruct.statsV2'
```

## Telling a real page from a challenge

When an IP is gated, TikTok returns a short firewall challenge page instead of
the data. `raw` prints whatever the page shipped, so a tiny blob with no detail
node is the sign you have been served the challenge rather than the content. The
typed commands detect this and exit 4 or 3; `raw` hands you the bytes so you can
see it yourself. See [troubleshooting](/reference/troubleshooting/) for the full
surface map.
