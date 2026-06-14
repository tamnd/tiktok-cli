---
title: "A video and its author"
description: "Read one video, then page the author's other posts."
weight: 10
---

The `video` command is the most reliable in the tool. It reads the JSON the
video page already ships, so it needs no signing and answers from anywhere.

```bash
tt video https://www.tiktok.com/@tiktok/video/7106594312292453675 -o json
```

The record carries the caption, the create time, the author, the sound, the
hashtags, the dimensions, the playable urls, and every counter. The author block
includes a `author_sec_uid`, the opaque id the posts endpoint needs.

## From one video to the author's feed

`posts` takes a handle or a secUid. Given a handle it resolves the secUid from
the profile page first, then pages the feed:

```bash
tt posts @tiktok -n 60 -o jsonl
```

If you already have a secUid from a video record, pass it directly to skip the
lookup:

```bash
tt posts MS4wLjABAAAAv7iSuuXDJGDvJkmH_vz1qkDZYo1apxgzaxdBSeIuPiM -n 60
```

`posts` rides the signed API plane, so from a datacenter IP it may exit 4. See
[troubleshooting](/reference/troubleshooting/).

## Just the links

Every record carries a `url`, so `-o url` gives a clean link stream:

```bash
tt posts @tiktok -n 100 -o url > links.txt
```

## Picking columns

`--fields` selects and orders columns for the table and the delimited formats:

```bash
tt posts @tiktok -o csv --fields id,create_time,play_count,digg_count
```
