package tiktok

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// discovery is the uniform, rankable row the graph walk emits. The full per-kind
// record stays reachable by id through the single-record commands.
type discovery struct {
	Kind   string  `json:"kind"`
	ID     string  `json:"id" kit:"id"`
	Name   string  `json:"name" table:"name,truncate"`
	Author string  `json:"author"`
	Metric int64   `json:"metric"`
	Score  float64 `json:"score"`
	Depth  int     `json:"depth"`
	Via    string  `json:"via"`
	URL    string  `json:"url"`
}

func toDiscovery(n Node) discovery {
	d := discovery{
		Kind:  n.Kind,
		ID:    n.ID,
		Score: math.Round(n.Score*10000) / 10000,
		Depth: n.Depth,
		Via:   n.Via,
	}
	switch n.Kind {
	case KindUser:
		if n.User != nil {
			d.Name = n.User.Nickname
			if d.Name == "" {
				d.Name = n.User.UniqueID
			}
			d.Author = n.User.UniqueID
			d.Metric = n.User.FollowerCount
			d.URL = n.User.URL
			if d.URL == "" && n.User.UniqueID != "" {
				d.URL = Host + "/@" + n.User.UniqueID
			}
		}
	case KindVideo:
		if n.Video != nil {
			d.Name = n.Video.Desc
			d.Author = n.Video.Author
			d.Metric = n.Video.PlayCount
			d.URL = n.Video.URL
		}
	case KindHashtag:
		if n.Hashtag != nil {
			d.Name = n.Hashtag.Title
			d.Metric = n.Hashtag.ViewCount
			d.URL = n.Hashtag.URL
		}
	case KindSound:
		if n.Sound != nil {
			d.Name = n.Sound.Title
			d.Author = n.Sound.AuthorName
			d.Metric = n.Sound.VideoCount
			d.URL = n.Sound.URL
		}
	}
	return d
}

type discoverIn struct {
	Seeds      []string `kit:"flag,name=seed" help:"seed a user by @handle (repeatable)"`
	Tags       []string `kit:"flag,name=seed-tag" help:"seed a hashtag by name (repeatable)"`
	SeedSound  string   `kit:"flag,name=seed-sound" help:"seed a sound by id or url"`
	SeedVideo  string   `kit:"flag,name=seed-video" help:"seed a single video by id or url"`
	SeedSearch string   `kit:"flag,name=seed-search" help:"seed from a search phrase"`
	Trending   bool     `kit:"flag" help:"seed from the trending feed"`

	Depth       int `kit:"flag" help:"maximum hops from a seed"`
	MaxNodes    int `kit:"flag,name=max-nodes" help:"stop after this many nodes (0 = built-in default)"`
	MaxRequests int `kit:"flag,name=max-requests" help:"stop after this many requests (0 = built-in default)"`
	Fanout      int `kit:"flag" help:"neighbors taken per list-bearing node (0 = built-in default)"`
	CommentMine int `kit:"flag,name=comment-mine" help:"commenters mined per video (0 = off)"`

	Top      int      `kit:"flag" help:"emit only the N highest-scored nodes"`
	MinScore float64  `kit:"flag,name=min-score" help:"drop nodes scored below this from the output"`
	Kinds    []string `kit:"flag,name=kind" help:"restrict emitted kinds: user,video,hashtag,sound"`
	Edges    string   `kit:"flag" help:"write the walked edges as JSONL to this file"`

	Sess *Session `kit:"inject"`
}

func registerDiscover(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "discover", Group: "read",
		Summary: "Walk the public graph from seeds and rank the hottest nodes it reaches",
		Long: `discover starts from one or more seeds and walks the public TikTok graph
outward, scoring each node it reaches by how hot it is (plays and velocity for
videos, followers for users, views for hashtags, uses for sounds). It emits a
ranked, uniform row per node.

Most expansion edges ride the signed API plane, which the firewall gates from
datacenter IPs, so a useful walk needs a residential session. From a gated IP
the walk runs correctly, reaches little, and says so: it exits 4 when the wall
stops it and prints a summary naming the walled surfaces.`,
	}, discover)
}

func discover(ctx context.Context, in discoverIn, emit func(discovery) error) error {
	var sd []Seed
	for _, s := range in.Seeds {
		sd = append(sd, Seed{Kind: SeedUser, Value: s})
	}
	for _, t := range in.Tags {
		sd = append(sd, Seed{Kind: SeedHashtag, Value: t})
	}
	if in.SeedSound != "" {
		sd = append(sd, Seed{Kind: SeedSound, Value: in.SeedSound})
	}
	if in.SeedVideo != "" {
		sd = append(sd, Seed{Kind: SeedVideo, Value: in.SeedVideo})
	}
	if in.SeedSearch != "" {
		sd = append(sd, Seed{Kind: SeedSearch, Value: in.SeedSearch})
	}
	if in.Trending {
		sd = append(sd, Seed{Kind: SeedTrending})
	}
	if len(sd) == 0 {
		return errs.Usage("need at least one seed: --seed, --seed-tag, --seed-sound, --seed-video, --seed-search, or --trending")
	}

	kindSet := map[string]bool{}
	for _, k := range in.Kinds {
		kindSet[strings.TrimSpace(k)] = true
	}
	want := func(k string) bool { return len(kindSet) == 0 || kindSet[k] }

	var edgeFile *os.File
	var edgeEnc *json.Encoder
	if in.Edges != "" {
		f, err := os.Create(in.Edges)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		edgeFile = f
		edgeEnc = json.NewEncoder(f)
	}

	cr := &Crawler{
		Src: in.Sess.Client,
		Opts: CrawlOptions{
			Depth:       in.Depth,
			MaxNodes:    in.MaxNodes,
			MaxRequests: in.MaxRequests,
			Fanout:      in.Fanout,
			CommentMine: in.CommentMine,
		},
	}

	var rows []discovery
	cr.OnNode = func(n Node) {
		if !want(n.Kind) {
			return
		}
		d := toDiscovery(n)
		if d.Score < in.MinScore {
			return
		}
		rows = append(rows, d)
	}
	if edgeEnc != nil {
		cr.OnEdge = func(e Edge) { _ = edgeEnc.Encode(e) }
	}

	in.Sess.Progressf("walking from %d seed(s), depth %d", len(sd), in.Depth)
	sum, err := cr.Run(ctx, sd)
	if err != nil {
		return MapErr(err)
	}
	if edgeFile != nil {
		_ = edgeFile.Sync()
	}

	// rows arrive hottest-first already; --top makes the cut exact.
	if in.Top > 0 {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
		if len(rows) > in.Top {
			rows = rows[:in.Top]
		}
	}

	for _, d := range rows {
		if err := emit(d); err != nil {
			return err
		}
	}

	discoverSummary(in.Sess, sum)

	// A walk that reached nothing because the wall stopped it is exit 4, so a
	// gated run is distinguishable from a genuinely empty one (exit 3, which kit
	// returns on its own when the stream stays empty).
	if len(rows) == 0 && sum.Walled {
		return errs.NeedAuth("%s", ErrWalled.Error())
	}
	return nil
}

// discoverSummary prints the honesty surface to stderr unless quiet.
func discoverSummary(sess *Session, s Summary) {
	if sess == nil || sess.Quiet {
		return
	}
	var b strings.Builder
	fmt.Fprintf(&b, "reached %d node(s)", s.NodesEmitted)
	if len(s.NodesByKind) > 0 {
		fmt.Fprintf(&b, " (%s)", joinCounts(s.NodesByKind))
	}
	fmt.Fprintf(&b, ", %d edge(s), %d request(s)", sumCounts(s.EdgesByType), s.Requests)
	if n := sumCounts(s.WalledEdges); n > 0 {
		fmt.Fprintf(&b, "; %d walled (%s)", n, joinCounts(s.WalledEdges))
	}
	fmt.Fprintf(&b, "; stopped: %s", s.StopReason)
	fmt.Fprintln(os.Stderr, b.String())
}

func sumCounts(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}

func joinCounts(m map[string]int) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s %d", k, m[k]))
	}
	return strings.Join(parts, ", ")
}
