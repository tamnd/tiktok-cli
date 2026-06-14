package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/tiktok-cli/tiktok"
)

// discovery is the uniform, rankable row the walk emits. The full per-kind
// record stays reachable by id through the single commands.
type discovery struct {
	Kind   string  `json:"kind"`
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Author string  `json:"author"`
	Metric int64   `json:"metric"`
	Score  float64 `json:"score"`
	Depth  int     `json:"depth"`
	Via    string  `json:"via"`
	URL    string  `json:"url"`
}

func toDiscovery(n tiktok.Node) discovery {
	d := discovery{
		Kind:  n.Kind,
		ID:    n.ID,
		Score: math.Round(n.Score*10000) / 10000,
		Depth: n.Depth,
		Via:   n.Via,
	}
	switch n.Kind {
	case tiktok.KindUser:
		if n.User != nil {
			d.Name = n.User.Nickname
			if d.Name == "" {
				d.Name = n.User.UniqueID
			}
			d.Author = n.User.UniqueID
			d.Metric = n.User.FollowerCount
			d.URL = n.User.URL
			if d.URL == "" && n.User.UniqueID != "" {
				d.URL = tiktok.Host + "/@" + n.User.UniqueID
			}
		}
	case tiktok.KindVideo:
		if n.Video != nil {
			d.Name = n.Video.Desc
			d.Author = n.Video.Author
			d.Metric = n.Video.PlayCount
			d.URL = n.Video.URL
		}
	case tiktok.KindHashtag:
		if n.Hashtag != nil {
			d.Name = n.Hashtag.Title
			d.Metric = n.Hashtag.ViewCount
			d.URL = n.Hashtag.URL
		}
	case tiktok.KindSound:
		if n.Sound != nil {
			d.Name = n.Sound.Title
			d.Author = n.Sound.AuthorName
			d.Metric = n.Sound.VideoCount
			d.URL = n.Sound.URL
		}
	}
	return d
}

func (a *App) discoverCmd() *cobra.Command {
	var (
		seeds      []string
		tags       []string
		seedSound  string
		seedVideo  string
		seedSearch string
		trending   bool

		depth       int
		maxNodes    int
		maxRequests int
		fanout      int
		commentMine int

		top      int
		minScore float64
		kinds    []string
		edges    string
	)

	cmd := &cobra.Command{
		Use:   "discover [flags]",
		Short: "Walk the public graph from seeds and rank the hottest nodes it reaches",
		Long: `discover starts from one or more seeds and walks the public TikTok graph
outward, scoring each node it reaches by how hot it is (plays and velocity for
videos, followers for users, views for hashtags, uses for sounds). It emits a
ranked, uniform row per node.

Most expansion edges ride the signed API plane, which the firewall gates from
datacenter IPs, so a useful walk needs a residential session. From a gated IP
the walk runs correctly, reaches little, and says so: it exits 4 when the wall
stops it and prints a summary naming the walled surfaces.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			var sd []tiktok.Seed
			for _, s := range seeds {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedUser, Value: s})
			}
			for _, t := range tags {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedHashtag, Value: t})
			}
			if seedSound != "" {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedSound, Value: seedSound})
			}
			if seedVideo != "" {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedVideo, Value: seedVideo})
			}
			if seedSearch != "" {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedSearch, Value: seedSearch})
			}
			if trending {
				sd = append(sd, tiktok.Seed{Kind: tiktok.SeedTrending})
			}
			if len(sd) == 0 {
				return codeError(exitUsage, errors.New("need at least one seed: --seed, --seed-tag, --seed-sound, --seed-video, --seed-search, or --trending"))
			}

			kindSet := map[string]bool{}
			for _, k := range kinds {
				kindSet[strings.TrimSpace(k)] = true
			}
			want := func(k string) bool { return len(kindSet) == 0 || kindSet[k] }

			var edgeFile *os.File
			var edgeEnc *json.Encoder
			if edges != "" {
				f, err := os.Create(edges)
				if err != nil {
					return codeError(exitError, err)
				}
				defer func() { _ = f.Close() }()
				edgeFile = f
				edgeEnc = json.NewEncoder(f)
			}

			cr := &tiktok.Crawler{
				Src: a.client,
				Opts: tiktok.CrawlOptions{
					Depth:       depth,
					MaxNodes:    maxNodes,
					MaxRequests: maxRequests,
					Fanout:      fanout,
					CommentMine: commentMine,
				},
			}

			var rows []discovery
			cr.OnNode = func(n tiktok.Node) {
				if !want(n.Kind) {
					return
				}
				d := toDiscovery(n)
				if d.Score < minScore {
					return
				}
				rows = append(rows, d)
			}
			if edgeEnc != nil {
				cr.OnEdge = func(e tiktok.Edge) { _ = edgeEnc.Encode(e) }
			}

			a.progressf("walking from %d seed(s), depth %d", len(sd), depth)
			sum, err := cr.Run(cmd.Context(), sd)
			if err != nil {
				return mapErr(err)
			}
			if edgeFile != nil {
				_ = edgeFile.Sync()
			}

			// rows arrive hottest-first already; --top makes the cut exact.
			if top > 0 {
				sort.SliceStable(rows, func(i, j int) bool { return rows[i].Score > rows[j].Score })
				if len(rows) > top {
					rows = rows[:top]
				}
			}

			r := NewRenderer(os.Stdout, Format(a.output), a.fields, a.noHeader, a.template)
			if err := r.Render(rows); err != nil {
				return codeError(exitError, err)
			}

			a.discoverSummary(sum)

			if len(rows) == 0 {
				if sum.Walled {
					return codeError(exitWalled, tiktok.ErrWalled)
				}
				return codeError(exitNoData, nil)
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringSliceVar(&seeds, "seed", nil, "seed a user by @handle (repeatable)")
	f.StringSliceVar(&tags, "seed-tag", nil, "seed a hashtag by name (repeatable)")
	f.StringVar(&seedSound, "seed-sound", "", "seed a sound by id or url")
	f.StringVar(&seedVideo, "seed-video", "", "seed a single video by id or url")
	f.StringVar(&seedSearch, "seed-search", "", "seed from a search phrase")
	f.BoolVar(&trending, "trending", false, "seed from the trending feed")

	f.IntVar(&depth, "depth", 2, "maximum hops from a seed")
	f.IntVar(&maxNodes, "max-nodes", 0, "stop after this many nodes (0 = built-in default)")
	f.IntVar(&maxRequests, "max-requests", 0, "stop after this many requests (0 = built-in default)")
	f.IntVar(&fanout, "fanout", 0, "neighbors taken per list-bearing node (0 = built-in default)")
	f.IntVar(&commentMine, "comment-mine", 0, "commenters mined per video (0 = off)")

	f.IntVar(&top, "top", 0, "emit only the N highest-scored nodes")
	f.Float64Var(&minScore, "min-score", 0, "drop nodes scored below this from the output")
	f.StringSliceVar(&kinds, "kind", nil, "restrict emitted kinds: user,video,hashtag,sound")
	f.StringVar(&edges, "edges", "", "write the walked edges as JSONL to this file")

	return cmd
}

// discoverSummary prints the honesty surface to stderr unless quiet.
func (a *App) discoverSummary(s tiktok.Summary) {
	if a.quiet {
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
