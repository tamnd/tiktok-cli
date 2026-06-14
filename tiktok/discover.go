package tiktok

import (
	"container/heap"
	"context"
	"errors"
	"math"
	"time"
)

// Node kinds carried by the discovery walk.
const (
	KindUser    = "user"
	KindVideo   = "video"
	KindHashtag = "hashtag"
	KindSound   = "sound"
)

// Source is the slice of the client the walk depends on. *Client satisfies it.
// Tests inject a fake to run the walk offline and deterministically.
type Source interface {
	VideoByID(ctx context.Context, author, id string) (Video, error)
	UserByHandle(ctx context.Context, handle string) (User, error)
	Posts(ctx context.Context, handle string, limit int, cursor string) ([]Video, error)
	HashtagByName(ctx context.Context, name string) (Hashtag, error)
	HashtagVideos(ctx context.Context, challengeID string, limit int) ([]Video, error)
	SoundByID(ctx context.Context, slug, id string) (Sound, error)
	SoundVideos(ctx context.Context, musicID string, limit int) ([]Video, error)
	Comments(ctx context.Context, videoID, author string, limit int) ([]Comment, error)
	Related(ctx context.Context, videoID string, limit int) ([]Video, error)
	Search(ctx context.Context, keyword string, limit int) ([]SearchHit, error)
	Trending(ctx context.Context, limit int) ([]Video, error)
}

// SeedKind is the kind of a starting point for a walk.
type SeedKind int

const (
	SeedUser SeedKind = iota
	SeedHashtag
	SeedSound
	SeedVideo
	SeedSearch
	SeedTrending
)

// Seed is one starting point. Value is empty for SeedTrending.
type Seed struct {
	Kind  SeedKind
	Value string
}

// Node is one reached graph node. Exactly one record pointer is set per Kind.
type Node struct {
	Kind    string
	ID      string
	Depth   int
	Via     string
	Score   float64
	User    *User
	Video   *Video
	Hashtag *Hashtag
	Sound   *Sound
}

// Edge records that expanding From led to To along Type.
type Edge struct {
	FromID   string `json:"from_id"`
	FromKind string `json:"from_kind"`
	ToID     string `json:"to_id"`
	ToKind   string `json:"to_kind"`
	Type     string `json:"type"`
}

// CrawlOptions bounds and tunes a walk. Zero values take sane defaults.
type CrawlOptions struct {
	Depth       int           // maximum hops from a seed
	MaxNodes    int           // stop after this many nodes are emitted
	MaxRequests int           // stop after this many source calls
	Deadline    time.Duration // wall-clock cap, 0 means off
	Fanout      int           // neighbors taken per list-bearing node
	CommentMine int           // commenters mined per video, 0 means off
	Decay       float64       // per-hop score decay applied to the frontier
	Now         func() time.Time
}

// Summary reports what a walk reached and why it stopped.
type Summary struct {
	NodesByKind  map[string]int
	EdgesByType  map[string]int
	WalledEdges  map[string]int
	Errors       map[string]int
	Requests     int
	NodesEmitted int
	Walled       bool
	StopReason   string
}

// Crawler walks the public graph from seeds, scoring as it goes.
type Crawler struct {
	Src    Source
	Opts   CrawlOptions
	OnNode func(Node) // called as each node with data is emitted
	OnEdge func(Edge) // optional, called as each edge is walked
}

func (c *Crawler) normalize() {
	if c.Opts.Depth <= 0 {
		c.Opts.Depth = 2
	}
	if c.Opts.MaxNodes <= 0 {
		c.Opts.MaxNodes = 500
	}
	if c.Opts.MaxRequests <= 0 {
		c.Opts.MaxRequests = 2000
	}
	if c.Opts.Fanout <= 0 {
		c.Opts.Fanout = 30
	}
	if c.Opts.Decay <= 0 {
		c.Opts.Decay = 0.85
	}
	if c.Opts.Now == nil {
		c.Opts.Now = time.Now
	}
}

func (c *Crawler) now() time.Time { return c.Opts.Now() }

// Run walks from the seeds and returns the run summary. It streams emitted
// nodes through OnNode and edges through OnEdge as it goes.
func (c *Crawler) Run(ctx context.Context, seeds []Seed) (Summary, error) {
	c.normalize()
	w := &walk{
		c:   c,
		ctx: ctx,
		fr:  &pq{},
		sum: Summary{
			NodesByKind: map[string]int{},
			EdgesByType: map[string]int{},
			WalledEdges: map[string]int{},
			Errors:      map[string]int{},
		},
		seen:     map[string]bool{},
		expanded: map[string]bool{},
		start:    c.now(),
	}
	heap.Init(w.fr)
	for _, s := range seeds {
		w.seedOne(s)
	}
	for w.fr.Len() > 0 {
		if reason, over := w.overBudget(); over {
			w.sum.StopReason = reason
			break
		}
		it := heap.Pop(w.fr).(*item)
		key := it.node.Kind + ":" + it.node.ID
		if w.expanded[key] {
			continue
		}
		w.expanded[key] = true

		node := w.materialize(it.node)
		if hasData(node) {
			node.Score = c.scoreNode(node)
			w.emit(node)
		}
		if node.Depth < c.Opts.Depth {
			w.expand(node)
		}
	}
	if w.sum.StopReason == "" {
		w.sum.StopReason = "frontier drained"
	}
	w.sum.Requests = w.reqs
	w.sum.NodesEmitted = w.emitted
	w.sum.Walled = w.emitted == 0 && sumMap(w.sum.WalledEdges) > 0
	return w.sum, nil
}

// walk holds the mutable state of a single Run.
type walk struct {
	c            *Crawler
	ctx          context.Context
	fr           *pq
	seen         map[string]bool
	expanded     map[string]bool
	reqs         int
	emitted      int
	walledStreak int
	sum          Summary
	start        time.Time
}

func (w *walk) overBudget() (string, bool) {
	o := w.c.Opts
	if o.MaxNodes > 0 && w.emitted >= o.MaxNodes {
		return "max-nodes", true
	}
	if o.MaxRequests > 0 && w.reqs >= o.MaxRequests {
		return "max-requests", true
	}
	if o.Deadline > 0 && w.c.now().Sub(w.start) >= o.Deadline {
		return "deadline", true
	}
	if w.emitted == 0 && w.walledStreak >= 8 {
		return "walled", true
	}
	return "", false
}

func (w *walk) emit(n Node) {
	w.emitted++
	w.sum.NodesByKind[n.Kind]++
	if w.c.OnNode != nil {
		w.c.OnNode(n)
	}
}

func (w *walk) recordErr(surface string, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, ErrWalled) {
		w.sum.WalledEdges[surface]++
		w.walledStreak++
		return
	}
	w.sum.Errors[surface]++
}

// seedOne pushes the depth-0 nodes a seed expands into.
func (w *walk) seedOne(s Seed) {
	switch s.Kind {
	case SeedUser:
		h, err := ParseHandle(s.Value)
		if err != nil {
			w.sum.Errors["seed-user"]++
			return
		}
		w.pushSeed(Node{Kind: KindUser, ID: userKey(h, ""), User: &User{UniqueID: h}}, "seed")
	case SeedHashtag:
		name := s.Value
		if len(name) > 0 && name[0] == '#' {
			name = name[1:]
		}
		w.pushSeed(Node{Kind: KindHashtag, ID: "name:" + name, Hashtag: &Hashtag{Title: name}}, "seed")
	case SeedSound:
		id, err := ParseMusicID(s.Value)
		if err != nil {
			w.sum.Errors["seed-sound"]++
			return
		}
		w.pushSeed(Node{Kind: KindSound, ID: id, Sound: &Sound{ID: id}}, "seed")
	case SeedVideo:
		id, err := ParseVideoID(s.Value)
		if err != nil {
			w.sum.Errors["seed-video"]++
			return
		}
		author, _ := ParseHandle(s.Value)
		w.pushSeed(Node{Kind: KindVideo, ID: id, Video: &Video{ID: id, Author: author}}, "seed")
	case SeedSearch:
		hits, err := w.search(s.Value, maxInt(w.c.Opts.Fanout, 20))
		if err != nil {
			w.recordErr("search", err)
			return
		}
		for _, h := range hits {
			switch h.Type {
			case "video":
				w.pushSeed(Node{Kind: KindVideo, ID: h.ID, Video: &Video{ID: h.ID, Author: h.Author, Desc: h.Title, URL: h.URL}}, "search_hit")
			case "user":
				handle, _ := ParseHandle(h.URL)
				w.pushSeed(Node{Kind: KindUser, ID: userKey(handle, ""), User: &User{UniqueID: handle, Nickname: h.Title}}, "search_hit")
			}
		}
	case SeedTrending:
		vids, err := w.trending(maxInt(w.c.Opts.Fanout, 30))
		if err != nil {
			w.recordErr("trending", err)
			return
		}
		for i := range vids {
			v := vids[i]
			w.pushSeed(Node{Kind: KindVideo, ID: v.ID, Video: &v}, "trending")
		}
	}
}

func (w *walk) pushSeed(n Node, via string) {
	n.Depth = 0
	n.Via = via
	key := n.Kind + ":" + n.ID
	if w.seen[key] {
		return
	}
	w.seen[key] = true
	n.Score = w.c.scoreNode(n)
	heap.Push(w.fr, &item{node: n, prio: 1.0}) // seeds expand first
}

// discover records an edge and enqueues a neighbor if it is new and in depth.
func (w *walk) discover(parent Node, edge string, neighbor Node) {
	neighbor.Depth = parent.Depth + 1
	neighbor.Via = edge
	w.sum.EdgesByType[edge]++
	if w.c.OnEdge != nil {
		w.c.OnEdge(Edge{
			FromID:   parent.ID,
			FromKind: parent.Kind,
			ToID:     neighbor.ID,
			ToKind:   neighbor.Kind,
			Type:     edge,
		})
	}
	if neighbor.Depth > w.c.Opts.Depth {
		return
	}
	key := neighbor.Kind + ":" + neighbor.ID
	if w.seen[key] {
		return
	}
	w.seen[key] = true
	neighbor.Score = w.c.scoreNode(neighbor)
	prio := neighbor.Score * math.Pow(w.c.Opts.Decay, float64(neighbor.Depth))
	heap.Push(w.fr, &item{node: neighbor, prio: prio})
}

// materialize fetches the authoritative record for a node when the discovery
// edge carried only a partial one. A walled fetch leaves the partial in place.
func (w *walk) materialize(n Node) Node {
	switch n.Kind {
	case KindVideo:
		if n.Video == nil || (n.Video.CreateTime == 0 && n.Video.PlayCount == 0 && n.Video.Desc == "") {
			author := ""
			if n.Video != nil {
				author = n.Video.Author
			}
			v, err := w.videoByID(author, n.ID)
			if err == nil {
				n.Video = &v
			} else {
				w.recordErr("video", err)
			}
		}
	case KindUser:
		if n.User == nil || n.User.ID == "" {
			handle := ""
			if n.User != nil {
				handle = n.User.UniqueID
			}
			if handle != "" {
				u, err := w.userByHandle(handle)
				if err == nil {
					if n.User != nil && u.SecUID == "" {
						u.SecUID = n.User.SecUID
					}
					n.User = &u
				} else {
					w.recordErr("user", err)
				}
			}
		}
	case KindHashtag:
		if n.Hashtag == nil || n.Hashtag.ID == "" {
			name := ""
			if n.Hashtag != nil {
				name = n.Hashtag.Title
			}
			if name != "" {
				h, err := w.hashtagByName(name)
				if err == nil {
					n.Hashtag = &h
				} else {
					w.recordErr("hashtag", err)
				}
			}
		}
	case KindSound:
		if n.Sound == nil || n.Sound.VideoCount == 0 {
			id := n.ID
			if n.Sound != nil && n.Sound.ID != "" {
				id = n.Sound.ID
			}
			if id != "" {
				s, err := w.soundByID(id)
				if err == nil {
					n.Sound = &s
				} else {
					w.recordErr("sound", err)
				}
			}
		}
	}
	return n
}

// expand yields the edges out of a node and enqueues their neighbors.
func (w *walk) expand(n Node) {
	switch n.Kind {
	case KindVideo:
		w.expandVideo(n)
	case KindUser:
		w.expandUser(n)
	case KindHashtag:
		w.expandHashtag(n)
	case KindSound:
		w.expandSound(n)
	}
}

func (w *walk) expandVideo(n Node) {
	v := n.Video
	if v == nil {
		// a stub seed video that walled, still try the API related edge.
		if w.c.Opts.Fanout > 0 {
			w.relatedEdge(n, n.ID)
		}
		return
	}
	if v.Author != "" {
		w.discover(n, "authored", Node{
			Kind: KindUser, ID: userKey(v.Author, v.AuthorSecUID),
			User: &User{UniqueID: v.Author, SecUID: v.AuthorSecUID},
		})
	}
	if v.MusicID != "" {
		w.discover(n, "uses_sound", Node{
			Kind: KindSound, ID: v.MusicID,
			Sound: &Sound{ID: v.MusicID, Title: v.MusicTitle, AuthorName: v.MusicAuthor},
		})
	}
	for _, t := range v.Challenges {
		w.discover(n, "tagged", Node{Kind: KindHashtag, ID: "name:" + t, Hashtag: &Hashtag{Title: t}})
	}
	for _, h := range mentions(v.Desc) {
		if h == v.Author {
			continue
		}
		w.discover(n, "mentions", Node{Kind: KindUser, ID: userKey(h, ""), User: &User{UniqueID: h}})
	}
	if w.c.Opts.Fanout > 0 {
		w.relatedEdge(n, v.ID)
	}
	if w.c.Opts.CommentMine > 0 {
		cs, err := w.comments(v.ID, v.Author, w.c.Opts.CommentMine)
		if err != nil {
			w.recordErr("comments", err)
		} else {
			seen := map[string]bool{}
			for _, cm := range cs {
				if cm.Author == "" || seen[cm.Author] {
					continue
				}
				seen[cm.Author] = true
				w.discover(n, "commented", Node{
					Kind: KindUser, ID: userKey(cm.Author, ""),
					User: &User{UniqueID: cm.Author, Nickname: cm.AuthorNick},
				})
			}
		}
	}
}

func (w *walk) relatedEdge(n Node, videoID string) {
	vids, err := w.related(videoID, w.c.Opts.Fanout)
	if err != nil {
		w.recordErr("related", err)
		return
	}
	for i := range vids {
		v := vids[i]
		w.discover(n, "related", Node{Kind: KindVideo, ID: v.ID, Video: &v})
	}
}

func (w *walk) expandUser(n Node) {
	u := n.User
	if u == nil {
		return
	}
	who := u.SecUID
	if who == "" {
		who = u.UniqueID
	}
	if who == "" {
		return
	}
	vids, err := w.posts(who, w.c.Opts.Fanout)
	if err != nil {
		w.recordErr("posts", err)
		return
	}
	for i := range vids {
		v := vids[i]
		w.discover(n, "posted", Node{Kind: KindVideo, ID: v.ID, Video: &v})
	}
}

func (w *walk) expandHashtag(n Node) {
	if n.Hashtag == nil || n.Hashtag.ID == "" {
		return
	}
	vids, err := w.hashtagVideos(n.Hashtag.ID, w.c.Opts.Fanout)
	if err != nil {
		w.recordErr("hashtag_videos", err)
		return
	}
	for i := range vids {
		v := vids[i]
		w.discover(n, "tagged", Node{Kind: KindVideo, ID: v.ID, Video: &v})
	}
}

func (w *walk) expandSound(n Node) {
	if n.Sound == nil || n.Sound.ID == "" {
		return
	}
	vids, err := w.soundVideos(n.Sound.ID, w.c.Opts.Fanout)
	if err != nil {
		w.recordErr("sound_videos", err)
		return
	}
	for i := range vids {
		v := vids[i]
		w.discover(n, "uses_sound", Node{Kind: KindVideo, ID: v.ID, Video: &v})
	}
}

// source wrappers count requests and reset the wall streak on success.

func (w *walk) videoByID(author, id string) (Video, error) {
	w.reqs++
	v, err := w.c.Src.VideoByID(w.ctx, author, id)
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

func (w *walk) userByHandle(h string) (User, error) {
	w.reqs++
	u, err := w.c.Src.UserByHandle(w.ctx, h)
	if err == nil {
		w.walledStreak = 0
	}
	return u, err
}

func (w *walk) posts(who string, n int) ([]Video, error) {
	w.reqs++
	v, err := w.c.Src.Posts(w.ctx, who, n, "")
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

func (w *walk) hashtagByName(name string) (Hashtag, error) {
	w.reqs++
	h, err := w.c.Src.HashtagByName(w.ctx, name)
	if err == nil {
		w.walledStreak = 0
	}
	return h, err
}

func (w *walk) hashtagVideos(id string, n int) ([]Video, error) {
	w.reqs++
	v, err := w.c.Src.HashtagVideos(w.ctx, id, n)
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

func (w *walk) soundByID(id string) (Sound, error) {
	w.reqs++
	s, err := w.c.Src.SoundByID(w.ctx, "", id)
	if err == nil {
		w.walledStreak = 0
	}
	return s, err
}

func (w *walk) soundVideos(id string, n int) ([]Video, error) {
	w.reqs++
	v, err := w.c.Src.SoundVideos(w.ctx, id, n)
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

func (w *walk) comments(id, author string, n int) ([]Comment, error) {
	w.reqs++
	c, err := w.c.Src.Comments(w.ctx, id, author, n)
	if err == nil {
		w.walledStreak = 0
	}
	return c, err
}

func (w *walk) related(id string, n int) ([]Video, error) {
	w.reqs++
	v, err := w.c.Src.Related(w.ctx, id, n)
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

func (w *walk) search(q string, n int) ([]SearchHit, error) {
	w.reqs++
	h, err := w.c.Src.Search(w.ctx, q, n)
	if err == nil {
		w.walledStreak = 0
	}
	return h, err
}

func (w *walk) trending(n int) ([]Video, error) {
	w.reqs++
	v, err := w.c.Src.Trending(w.ctx, n)
	if err == nil {
		w.walledStreak = 0
	}
	return v, err
}

// scoring

func (c *Crawler) scoreNode(n Node) float64 {
	switch n.Kind {
	case KindVideo:
		if n.Video != nil && (n.Video.PlayCount > 0 || n.Video.CreateTime > 0) {
			return videoScore(*n.Video, c.now())
		}
	case KindUser:
		if n.User != nil && n.User.ID != "" {
			return userScore(*n.User)
		}
	case KindHashtag:
		if n.Hashtag != nil && n.Hashtag.ID != "" {
			return hashtagScore(*n.Hashtag)
		}
	case KindSound:
		if n.Sound != nil && n.Sound.VideoCount > 0 {
			return soundScore(*n.Sound)
		}
	}
	return 0.4 // a partial record: worth exploring, below a known-hot signal
}

func videoScore(v Video, now time.Time) float64 {
	plays := float64(v.PlayCount)
	ageHours := 1.0
	if v.CreateTime > 0 {
		ageHours = math.Max(now.Sub(time.Unix(v.CreateTime, 0)).Hours(), 1)
	}
	velocity := plays / ageHours
	s := 0.6*lg(plays)/9 + 0.4*lg(velocity)/6
	return clamp01(s)
}

func userScore(u User) float64       { return clamp01(lg(float64(u.FollowerCount)) / 9) }
func hashtagScore(h Hashtag) float64 { return clamp01(lg(float64(h.ViewCount)) / 11) }
func soundScore(s Sound) float64     { return clamp01(lg(float64(s.VideoCount)) / 8) }

func lg(x float64) float64 {
	if x <= 0 {
		return 0
	}
	return math.Log10(x + 1)
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// hasData reports whether a node carries enough to be worth emitting.
func hasData(n Node) bool {
	switch n.Kind {
	case KindVideo:
		return n.Video != nil && n.Video.ID != "" &&
			(n.Video.Author != "" || n.Video.PlayCount > 0 || n.Video.CreateTime > 0 || n.Video.Desc != "")
	case KindUser:
		return n.User != nil && n.User.ID != ""
	case KindHashtag:
		return n.Hashtag != nil && n.Hashtag.ID != ""
	case KindSound:
		return n.Sound != nil && (n.Sound.VideoCount > 0 || n.Sound.AuthorName != "" || n.Sound.Title != "")
	}
	return false
}

// mentions pulls the @handles out of a caption.
func mentions(desc string) []string {
	var out []string
	for _, m := range reHandle.FindAllStringSubmatch(desc, -1) {
		out = append(out, m[1])
	}
	return out
}

// userKey keys a user by secUid when known, else by handle, so the seen-set
// dedups the common cases.
func userKey(handle, sec string) string {
	if sec != "" {
		return sec
	}
	return "handle:" + handle
}

func sumMap(m map[string]int) int {
	s := 0
	for _, v := range m {
		s += v
	}
	return s
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// pq is a max-heap of frontier items ordered by priority.
type item struct {
	node  Node
	prio  float64
	index int
}

type pq []*item

func (p pq) Len() int           { return len(p) }
func (p pq) Less(i, j int) bool { return p[i].prio > p[j].prio }
func (p pq) Swap(i, j int)      { p[i], p[j] = p[j], p[i]; p[i].index = i; p[j].index = j }
func (p *pq) Push(x any) {
	it := x.(*item)
	it.index = len(*p)
	*p = append(*p, it)
}
func (p *pq) Pop() any {
	old := *p
	n := len(old)
	it := old[n-1]
	old[n-1] = nil
	*p = old[:n-1]
	return it
}
