---
title: "Quick start"
description: "Run your first tt command."
weight: 30
---

Once `tt` is on your `PATH`, the most reliable command is `video`, which reads a
single post straight from the page:

```bash
tt video https://www.tiktok.com/@tiktok/video/7106594312292453675 -o json
```

That returns one record with the caption, author, sound, hashtags, dimensions,
and every counter, ready to pipe into `jq`.

A few more:

```bash
tt user tiktok                       # a profile
tt posts @tiktok -n 30               # a user's videos
tt comments 7106594312292453675 --author tiktok
tt hashtag minecraft --videos -n 50  # videos under a hashtag
tt sound 7106594280055130923 --videos
tt search "study with me" -n 20
tt trending -n 30
```

## Output that pipes

The default is a readable table on a terminal and JSONL when piped. Pick a
format with `-o`:

```bash
tt posts @tiktok -o jsonl | jq -r '.url'
tt video 7106594312292453675 --author tiktok -o csv --fields id,desc,play_count
tt trending -o url
```

## The two planes

`tt video`, `tt hashtag`, `tt sound`, and `tt raw` read the JSON a logged-out
page already ships, so they need no signing and answer reliably. `tt posts`,
`tt comments`, `tt search`, and `tt trending` call the signed web API, which
sits behind a firewall that scores the caller. From a datacenter IP those calls
are often gated, and `tt` exits 4 with a clear message when that happens. See
[troubleshooting](/reference/troubleshooting/) for the detail.
