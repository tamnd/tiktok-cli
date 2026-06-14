---
title: "Comments and replies"
description: "Read a video's comments, expand the threads, and pull replies under one comment."
weight: 40
---

A video's discussion comes in two layers. `comments` reads the top-level
comments under a video, and `replies` reads the answers under one of them. Both
ride the signed API plane, so from a datacenter IP they may exit 4. See
[troubleshooting](/reference/troubleshooting/).

## Top-level comments

Pass a video url or a bare id:

```bash
tt comments https://www.tiktok.com/@tiktok/video/7106594312292453675
tt comments 7106594312292453675 --author tiktok
```

A bare id has no author in it, so the record's `url` cannot be exact. Pass
`--author` to fill it in. Each comment record carries the text, the author, the
create time, the digg count, and a `reply_count` that tells you whether a thread
hangs under it:

```bash
tt comments 7106594312292453675 --author tiktok -o jsonl \
  | jq -r '[.author, .digg_count, .text] | @tsv'
```

## Expand every thread inline

`--replies` walks each comment that has answers and emits those replies right
after their parent, in one stream. A reply carries the parent's id in
`parent_id`, so you can tell the layers apart:

```bash
tt comments 7106594312292453675 --author tiktok --replies -o jsonl \
  | jq -r 'select(.parent_id != "") | .text'
```

## Replies under one comment

When you only want the answers under a single comment, `replies` takes the video
and the comment id:

```bash
tt replies 7106594312292453675 7107000000000000000 --author tiktok
```

The first argument is the video (url or id), the second is the comment id from a
`comments` record. The `--author` note is the same: pass it with a bare video id
so the urls come out exact.

## Top commenters at a glance

Comment records pipe into `jq` like any other. The most-digged comments on a
video:

```bash
tt comments 7106594312292453675 --author tiktok -n 200 -o jsonl \
  | jq -s 'sort_by(-.digg_count) | .[:10] | .[] | "\(.digg_count)\t\(.text)"' -r
```
