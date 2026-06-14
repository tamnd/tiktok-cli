---
title: "The trending feed"
description: "Read the logged-out recommend feed and slice it with the usual tools."
weight: 60
---

`trending` reads the feed TikTok serves a logged-out visitor, the same set of
videos the front page recommends with no account and no search. It returns video
records, the same shape `video` and `posts` return:

```bash
tt trending -n 30
```

It rides the signed API plane, so from a datacenter IP it may exit 4. See
[troubleshooting](/reference/troubleshooting/).

## Slice it

The records are ordinary video records, so the usual flags and `jq` apply. The
most-played of the current feed:

```bash
tt trending -n 50 -o jsonl | jq -s 'sort_by(-.play_count) | .[:10] | .[].url' -r
```

The authors who show up most often:

```bash
tt trending -n 100 -o jsonl | jq -r .author | sort | uniq -c | sort -rn | head
```

Just the links, to hand to another tool:

```bash
tt trending -n 50 -o url
```

## As a discovery seed

The feed is also a starting point for the graph walk. `tt discover --trending`
seeds the walk from the trending videos and ranks the hottest nodes around them.
See [discovering hot nodes](/guides/discovering-hot-nodes/).

```bash
tt discover --trending --top 25 -o table
```
