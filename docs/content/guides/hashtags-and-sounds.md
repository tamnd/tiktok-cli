---
title: "Hashtags and sounds"
description: "Read a hashtag or a sound, then the videos under it."
weight: 20
---

A hashtag (TikTok calls it a challenge) and a sound both have a header record and
a video list. The header reads from the page's own JSON; the video list rides
the signed API plane.

## A hashtag

The header record carries the tag id, the description, and the view and video
counts:

```bash
tt hashtag minecraft -o json
```

Add `--videos` to page the videos that use the tag instead:

```bash
tt hashtag minecraft --videos -n 50 -o jsonl
```

## A sound

Same shape. The header carries the title, the artist credit, whether it is an
original sound, and how many videos use it:

```bash
tt sound 7106594280055130923 -o json
tt sound 7106594280055130923 --videos -n 50
```

You can pass a full `/music/{slug}-{id}` url in place of the bare id.

## Mining a trend

Chain the pieces with `jq`. For example, the top authors using a hashtag:

```bash
tt hashtag minecraft --videos -n 200 -o jsonl \
  | jq -r '.author' | sort | uniq -c | sort -rn | head
```

The video list commands ride the API plane, so they may exit 4 from a datacenter
IP. The header records read the page directly, but TikTok ships the tag and music
pages with the detail node stripped out for some callers, so from a datacenter IP
the header can come back empty (exit 3) even though the page loaded. From a
residential session both the header and, with `--videos`, the list answer. The
`video` and `raw` commands are the two that come back from anywhere.
