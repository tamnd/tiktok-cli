---
title: "Searching and profiles"
description: "Search for videos and accounts, then read a profile in full."
weight: 50
---

Three commands cover finding things by words and reading an account. `search`
returns mixed hits, `users` narrows to accounts, and `user` reads one profile in
full. `search` and `users` ride the signed API plane and may exit 4 from a
datacenter IP. `user` reads the profile page's own JSON, which answers from many
IPs but is sometimes gated too. See [troubleshooting](/reference/troubleshooting/).

## Search for videos and accounts

`search` returns a mixed stream of video and account hits. Each hit carries a
`type` (`video` or `user`), an `id`, a `title`, an `author`, and a `url`:

```bash
tt search "study with me" -n 20
tt search study with me -n 20      # quotes are optional, words join
```

The `type` field lets you split the stream after the fact:

```bash
tt search "lofi" -n 50 -o jsonl | jq -r 'select(.type == "user") | .url'
```

## Narrow to accounts

When you only want accounts, `users` skips the videos and returns full profile
records, the same shape `user` returns:

```bash
tt users "news" -n 20
tt users news -o csv --fields unique_id,nickname,follower_count,verified
```

## Read one profile

`user` takes a handle, with or without the `@`, or a full profile url. It returns
one record with the nickname, the bio, the region, the verified and private
flags, and the follower, following, heart, and video counts:

```bash
tt user tiktok
tt user @tiktok -o json
tt user https://www.tiktok.com/@tiktok -o json
```

The record's `sec_uid` is the opaque id the [posts feed](/guides/a-video-and-its-author/)
takes directly, so you can go from a profile straight to its videos without a
second lookup:

```bash
secuid=$(tt user tiktok -o json | jq -r .sec_uid)
tt posts "$secuid" -n 60 -o jsonl
```

## From a search to the feeds

Because every hit and profile carries a `url` and an id, a search result flows
into the single commands. Find accounts, then page the first one's videos:

```bash
first=$(tt users "minecraft" -n 1 -o json | jq -r .unique_id)
tt posts "@$first" -n 30 -o url
```
