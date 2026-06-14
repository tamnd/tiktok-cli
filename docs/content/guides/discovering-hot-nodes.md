---
title: "Discovering hot nodes"
description: "Walk the public graph from seeds and rank the hottest users, videos, hashtags, and sounds it reaches."
weight: 30
---

The single commands answer "tell me about this one thing." `discover` answers
"start here and show me what is hot nearby." It walks the public graph outward
from one or more seeds, scores every node it reaches, and emits a ranked, uniform
row per node.

```bash
tt discover --seed-video https://www.tiktok.com/@tiktok/video/7106594312292453675 -o table
```

Each row is one node:

```
KIND   ID                   NAME              AUTHOR  METRIC    SCORE   DEPTH  VIA       URL
video  7106594312292453675  how many frogs..  tiktok  562500    0.4653  0      seed      https://...
user   MS4wLjABAA...         TikTok            tiktok  94400000  0.8861  1      authored  https://www.tiktok.com/@tiktok
sound  7106594280055130923  original sound    TikTok  0         0.4     1      uses_sound
```

`KIND` is one of `user`, `video`, `hashtag`, `sound`. `METRIC` is that kind's
headline number (followers, plays, views, uses). `SCORE` is a 0..1 hotness rank.
`DEPTH` is hops from a seed, and `VIA` is the edge the walk took to reach it.
Every row keeps a `url`, so the full record is one `tt video`/`tt user`/... away.

## Seeds

Start from anything. Seeds combine, so a single walk can fan out from several
roots at once:

```bash
tt discover \
  --seed @tiktok --seed @nasa \
  --seed-tag minecraft \
  --seed-video 7106594312292453675 \
  --seed-sound 7106594280055130923 \
  --seed-search "frog" \
  --trending
```

| Flag | Seeds from |
| --- | --- |
| `--seed @handle` | a user (repeatable) |
| `--seed-tag name` | a hashtag (repeatable) |
| `--seed-sound id` | a sound, by id or `/music/` url |
| `--seed-video id` | one video, by id or url |
| `--seed-search "phrase"` | a search result page |
| `--trending` | the trending feed |

At least one seed is required.

## How far it walks

`--depth` caps the hops from a seed (default 2). The walk is best-first: it
always expands the hottest unexpanded node next, so a shallow run still surfaces
the strongest nodes first. Bounds keep it honest:

```bash
tt discover --seed @tiktok --depth 3 --fanout 20 --max-nodes 200 --max-requests 800
```

- `--fanout` is how many neighbors it takes from each list-bearing node.
- `--max-nodes` stops after that many nodes are emitted.
- `--max-requests` stops after that many fetches.
- `--comment-mine N` additionally pulls up to `N` commenters per video as user
  nodes (off by default, and it rides the API plane).

## Ranking and filtering

The rows arrive hottest-first. `--top N` keeps only the N highest-scored, and
`--min-score` drops weak nodes:

```bash
tt discover --trending --top 25 --min-score 0.5 -o table
```

`--kind` restricts what gets emitted without changing what the walk explores, so
you can crawl through videos but print only the users it found:

```bash
tt discover --seed-tag minecraft --kind user --top 50 -o jsonl
```

## The edges

Add `--edges path.jsonl` to record the graph the walk traversed, one edge per
line, alongside the node output:

```bash
tt discover --seed-video 7106594312292453675 --edges edges.jsonl -o jsonl
```

```json
{"from_id":"7106594312292453675","from_kind":"video","to_id":"MS4w...","to_kind":"user","type":"authored"}
{"from_id":"7106594312292453675","from_kind":"video","to_id":"7106594280055130923","to_kind":"sound","type":"uses_sound"}
{"from_id":"7106594312292453675","from_kind":"video","to_id":"name:Minecraft","to_kind":"hashtag","type":"tagged"}
{"from_id":"7106594312292453675","from_kind":"video","to_id":"handle:Gorillo","to_kind":"user","type":"mentions"}
```

Feed both streams into a graph tool, or join them later by id.

## What a datacenter IP can and cannot reach

This matters more for `discover` than for any single command, because the walk
chains many surfaces together. Two of them carry the weight:

- The **page blobs** (a video's own JSON, a profile's own JSON) need no signing
  and answer from anywhere. They give the seed video, its author, its mentioned
  users, its sound, and its hashtags.
- The **list endpoints** (a user's posts, a video's related feed, a hashtag's or
  sound's videos, search) ride the signed API plane. TikTok's firewall gates that
  plane from datacenter IPs.

So from a datacenter IP a `--seed-video` walk still reaches the small
constellation around that video through the page blob, but it cannot page a
feed to go wider. The walk runs correctly and says exactly what it could not
reach. The stderr summary names the walled surfaces:

```
reached 4 node(s) (sound 1, user 2, video 1), 4 edge(s), 6 request(s); 1 walled (related 1); stopped: frontier drained
```

When the wall stops the walk cold (every seed needs the API plane and nothing
came back), `discover` exits 4. From a residential session the list endpoints
answer and the walk goes as wide as the bounds allow. See
[troubleshooting](/reference/troubleshooting/) for the full surface map.
