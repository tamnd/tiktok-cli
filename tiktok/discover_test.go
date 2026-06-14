package tiktok

import (
	"context"
	"testing"
	"time"
)

// fakeSource is a small fixed graph for offline, deterministic walk tests.
//
//	alice (1e6 followers) posts v1, v2
//	bob   (1e3 followers) posts v3
//	v1 by alice: plays 1e6, sound m1, tag "fun", mentions @bob
//	v2 by alice: plays 10
//	v3 by bob:   plays 5e5
//	hashtag "fun" (id H1, 1e9 views) -> v3
//	sound m1 (1000 videos) -> v1
type fakeSource struct {
	walledAPI bool // every signed surface returns ErrWalled
	calls     map[string]int
}

func newFake() *fakeSource { return &fakeSource{calls: map[string]int{}} }

func (f *fakeSource) bump(name string) { f.calls[name]++ }

var (
	fxAlice = User{ID: "1", UniqueID: "alice", Nickname: "Alice", SecUID: "SU_ALICE", FollowerCount: 1_000_000, URL: "https://www.tiktok.com/@alice"}
	fxBob   = User{ID: "2", UniqueID: "bob", Nickname: "Bob", SecUID: "SU_BOB", FollowerCount: 1_000, URL: "https://www.tiktok.com/@bob"}

	fxV1 = Video{ID: "v1", Desc: "frogs @bob", CreateTime: 1_700_000_000, Author: "alice", AuthorSecUID: "SU_ALICE", MusicID: "m1", MusicTitle: "beat", Challenges: []string{"fun"}, PlayCount: 1_000_000, URL: "https://www.tiktok.com/@alice/video/v1"}
	fxV2 = Video{ID: "v2", Desc: "quiet", CreateTime: 1_700_000_000, Author: "alice", AuthorSecUID: "SU_ALICE", PlayCount: 10, URL: "https://www.tiktok.com/@alice/video/v2"}
	fxV3 = Video{ID: "v3", Desc: "bob clip", CreateTime: 1_700_000_000, Author: "bob", AuthorSecUID: "SU_BOB", PlayCount: 500_000, URL: "https://www.tiktok.com/@bob/video/v3"}

	fxFun = Hashtag{ID: "H1", Title: "fun", ViewCount: 1_000_000_000, URL: "https://www.tiktok.com/tag/fun"}
	fxM1  = Sound{ID: "m1", Title: "beat", VideoCount: 1000, URL: "https://www.tiktok.com/music/beat-m1"}
)

func (f *fakeSource) VideoByID(_ context.Context, _, id string) (Video, error) {
	f.bump("video")
	switch id {
	case "v1":
		return fxV1, nil
	case "v2":
		return fxV2, nil
	case "v3":
		return fxV3, nil
	}
	return Video{}, ErrNotFound
}

func (f *fakeSource) UserByHandle(_ context.Context, h string) (User, error) {
	f.bump("user")
	switch h {
	case "alice":
		return fxAlice, nil
	case "bob":
		return fxBob, nil
	}
	return User{}, ErrNotFound
}

func (f *fakeSource) Posts(_ context.Context, who string, _ int, _ string) ([]Video, error) {
	f.bump("posts")
	if f.walledAPI {
		return nil, ErrWalled
	}
	switch who {
	case "SU_ALICE", "alice":
		return []Video{fxV1, fxV2}, nil
	case "SU_BOB", "bob":
		return []Video{fxV3}, nil
	}
	return nil, nil
}

func (f *fakeSource) HashtagByName(_ context.Context, name string) (Hashtag, error) {
	f.bump("hashtagByName")
	if name == "fun" {
		return fxFun, nil
	}
	return Hashtag{}, ErrNotFound
}

func (f *fakeSource) HashtagVideos(_ context.Context, id string, _ int) ([]Video, error) {
	f.bump("hashtagVideos")
	if f.walledAPI {
		return nil, ErrWalled
	}
	if id == "H1" {
		return []Video{fxV3}, nil
	}
	return nil, nil
}

func (f *fakeSource) SoundByID(_ context.Context, _, id string) (Sound, error) {
	f.bump("soundByID")
	if id == "m1" {
		return fxM1, nil
	}
	return Sound{}, ErrNotFound
}

func (f *fakeSource) SoundVideos(_ context.Context, id string, _ int) ([]Video, error) {
	f.bump("soundVideos")
	if f.walledAPI {
		return nil, ErrWalled
	}
	if id == "m1" {
		return []Video{fxV1}, nil
	}
	return nil, nil
}

func (f *fakeSource) Comments(_ context.Context, _, _ string, _ int) ([]Comment, error) {
	f.bump("comments")
	if f.walledAPI {
		return nil, ErrWalled
	}
	return nil, nil
}

func (f *fakeSource) Related(_ context.Context, _ string, _ int) ([]Video, error) {
	f.bump("related")
	if f.walledAPI {
		return nil, ErrWalled
	}
	return nil, nil
}

func (f *fakeSource) Search(_ context.Context, _ string, _ int) ([]SearchHit, error) {
	f.bump("search")
	if f.walledAPI {
		return nil, ErrWalled
	}
	return []SearchHit{
		{Type: "user", ID: "1", Title: "Alice", URL: "https://www.tiktok.com/@alice"},
		{Type: "video", ID: "v3", Title: "bob clip", Author: "bob", URL: fxV3.URL},
	}, nil
}

func (f *fakeSource) Trending(_ context.Context, _ int) ([]Video, error) {
	f.bump("trending")
	if f.walledAPI {
		return nil, ErrWalled
	}
	return []Video{fxV1, fxV3}, nil
}

// fixedNow pins the clock so the velocity term is deterministic.
func fixedNow() time.Time { return time.Unix(1_700_100_000, 0) }

func collect(t *testing.T, cr *Crawler, seeds ...Seed) ([]Node, Summary) {
	t.Helper()
	var nodes []Node
	cr.OnNode = func(n Node) { nodes = append(nodes, n) }
	sum, err := cr.Run(context.Background(), seeds)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return nodes, sum
}

func hasNode(nodes []Node, kind, id string) bool {
	for _, n := range nodes {
		if n.Kind == kind && n.ID == id {
			return true
		}
	}
	return false
}

func TestDiscoverSeedUserDepth1(t *testing.T) {
	cr := &Crawler{Src: newFake(), Opts: CrawlOptions{Depth: 1, Fanout: 10, Now: fixedNow}}
	nodes, sum := collect(t, cr, Seed{Kind: SeedUser, Value: "alice"})

	// a user seeded by handle keys on the handle: secUid is only known after
	// the profile is fetched, but the node id is fixed when the seed is pushed.
	if !hasNode(nodes, KindUser, "handle:alice") {
		t.Fatalf("expected alice user node, got %v", nodes)
	}
	if !hasNode(nodes, KindVideo, "v1") || !hasNode(nodes, KindVideo, "v2") {
		t.Fatalf("expected alice's videos v1 and v2, got %v", nodes)
	}
	// depth 1: videos are emitted but not expanded, so the sound and hashtag
	// under v1 never appear.
	if hasNode(nodes, KindSound, "m1") {
		t.Fatalf("did not expect depth-2 sound at depth 1")
	}
	if sum.NodesByKind[KindVideo] != 2 || sum.NodesByKind[KindUser] != 1 {
		t.Fatalf("kind counts: %v", sum.NodesByKind)
	}
}

func TestDiscoverSeedUserDepth2(t *testing.T) {
	cr := &Crawler{Src: newFake(), Opts: CrawlOptions{Depth: 2, Fanout: 10, Now: fixedNow}}
	nodes, _ := collect(t, cr, Seed{Kind: SeedUser, Value: "alice"})

	// v1 expands at depth 1 into its SSR edges at depth 2.
	if !hasNode(nodes, KindSound, "m1") {
		t.Fatalf("expected sound m1 from v1, got %v", nodes)
	}
	if !hasNode(nodes, KindHashtag, "name:fun") {
		t.Fatalf("expected hashtag fun from v1, got %v", nodes)
	}
	if !hasNode(nodes, KindUser, "handle:bob") {
		t.Fatalf("expected bob via @mention from v1, got %v", nodes)
	}
}

func TestDiscoverDedup(t *testing.T) {
	// Trending seeds v1 and v3; v3 is also reachable from the hashtag under v1.
	// The seen-set must keep v3 to one emission.
	cr := &Crawler{Src: newFake(), Opts: CrawlOptions{Depth: 3, Fanout: 10, Now: fixedNow}}
	nodes, _ := collect(t, cr, Seed{Kind: SeedTrending})

	count := 0
	for _, n := range nodes {
		if n.Kind == KindVideo && n.ID == "v3" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("v3 emitted %d times, want 1", count)
	}
}

func TestDiscoverBestFirstOrder(t *testing.T) {
	// Seeds all share priority 1.0, so best-first shows among discovered
	// neighbors. alice posts v1 (1e6 plays) and v2 (10 plays) at depth 1; the
	// frontier must pop and emit the hotter v1 before v2.
	cr := &Crawler{Src: newFake(), Opts: CrawlOptions{Depth: 2, Fanout: 10, Now: fixedNow}}
	nodes, _ := collect(t, cr, Seed{Kind: SeedUser, Value: "alice"})

	idx := func(id string) int {
		for i, n := range nodes {
			if n.Kind == KindVideo && n.ID == id {
				return i
			}
		}
		return -1
	}
	i1, i2 := idx("v1"), idx("v2")
	if i1 < 0 || i2 < 0 {
		t.Fatalf("want both videos emitted, got v1=%d v2=%d", i1, i2)
	}
	if i1 > i2 {
		t.Fatalf("hot v1 emitted after cold v2 (v1=%d v2=%d)", i1, i2)
	}
}

func TestDiscoverMaxNodes(t *testing.T) {
	cr := &Crawler{Src: newFake(), Opts: CrawlOptions{Depth: 3, Fanout: 10, MaxNodes: 2, Now: fixedNow}}
	nodes, sum := collect(t, cr, Seed{Kind: SeedUser, Value: "alice"})
	if len(nodes) != 2 {
		t.Fatalf("emitted %d nodes, want 2", len(nodes))
	}
	if sum.StopReason != "max-nodes" {
		t.Fatalf("stop reason %q, want max-nodes", sum.StopReason)
	}
}

func TestDiscoverWalledExit(t *testing.T) {
	// API plane walled and the only seed is a search, which is API. The walk
	// reaches nothing and reports walled.
	f := newFake()
	f.walledAPI = true
	cr := &Crawler{Src: f, Opts: CrawlOptions{Depth: 2, Fanout: 10, Now: fixedNow}}
	nodes, sum := collect(t, cr, Seed{Kind: SeedSearch, Value: "cats"})
	if len(nodes) != 0 {
		t.Fatalf("expected nothing reached, got %v", nodes)
	}
	if !sum.Walled {
		t.Fatalf("expected Walled summary, got %+v", sum)
	}
	if sum.WalledEdges["search"] == 0 {
		t.Fatalf("expected a walled search edge, got %v", sum.WalledEdges)
	}
}

func TestDiscoverWalledDegrades(t *testing.T) {
	// SSR user seed works, but the posts feed walls. Alice is still emitted
	// from her profile; the walk does not crash, it records the wall.
	f := newFake()
	f.walledAPI = true
	cr := &Crawler{Src: f, Opts: CrawlOptions{Depth: 2, Fanout: 10, Now: fixedNow}}
	nodes, sum := collect(t, cr, Seed{Kind: SeedUser, Value: "alice"})
	if !hasNode(nodes, KindUser, "handle:alice") {
		t.Fatalf("expected alice from her profile, got %v", nodes)
	}
	if sum.WalledEdges["posts"] == 0 {
		t.Fatalf("expected a walled posts edge, got %v", sum.WalledEdges)
	}
}

func TestVideoScoreOrders(t *testing.T) {
	now := fixedNow()
	hot := videoScore(fxV1, now)  // 1e6 plays
	cold := videoScore(fxV2, now) // 10 plays
	if hot <= cold {
		t.Fatalf("hot video scored %f, cold %f", hot, cold)
	}
	if hot < 0 || hot > 1 {
		t.Fatalf("score out of range: %f", hot)
	}
}

func TestClientSatisfiesSource(t *testing.T) {
	var _ Source = (*Client)(nil)
}
