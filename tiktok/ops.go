package tiktok

import (
	"context"
	"strings"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// ops.go declares the record-stream commands as kit operations. Each one is
// declared once and exposed as a CLI subcommand, an HTTP route, and an MCP tool;
// kit renders the records in every format, applies --limit, tees them into --db,
// and (for the URI-tagged ops) lets a host dereference them by tiktok:// URI.
//
// The single-record reads (user, video, hashtag, sound) carry URI metadata so a
// host can dereference tiktok://user/<handle> and friends. The list reads
// (posts, comments) are members of a parent resource, so they answer `ant ls`.
// The raw byte dump, the version banner, and the graph walk do not fit the
// emit-records shape; raw and version stay as escape-hatch commands in the cli
// package, and discover lives in discover_op.go.
func registerOps(app *kit.App) {
	registerUser(app)
	registerVideo(app)
	registerPosts(app)
	registerComments(app)
	registerReplies(app)
	registerSearch(app)
	registerUsers(app)
	registerHashtag(app)
	registerSound(app)
	registerTrending(app)
	registerDiscover(app)
}

// effectiveLimit picks the per-command default when -n is unset.
func effectiveLimit(n, def int) int {
	if n > 0 {
		return n
	}
	return def
}

// --- user ---

type userIn struct {
	Ref  string   `kit:"arg" help:"@handle or profile url"`
	Sess *Session `kit:"inject"`
}

func registerUser(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "user", Group: "read", Single: true,
		Summary:  "Profile record for a @handle",
		URIType:  "user",
		Resolver: true,
		Args:     []kit.Arg{{Name: "handle", Help: "@handle or profile url"}},
	}, func(ctx context.Context, in userIn, emit func(User) error) error {
		handle, err := ParseHandle(in.Ref)
		if err != nil {
			return errs.Usage("%s", err.Error())
		}
		in.Sess.Progressf("fetching profile @%s", handle)
		u, err := in.Sess.Client.UserByHandle(ctx, handle)
		if err != nil {
			return MapErr(err)
		}
		return emit(u)
	})
}

// --- video ---

type videoIn struct {
	Ref    string   `kit:"arg" help:"video url or id"`
	Author string   `kit:"flag" help:"author handle, used to build the canonical url for a bare id"`
	Sess   *Session `kit:"inject"`
}

func registerVideo(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "video", Group: "read", Single: true,
		Summary:  "One video record",
		URIType:  "video",
		Resolver: true,
		Args:     []kit.Arg{{Name: "url-or-id", Help: "video url or numeric id"}},
	}, func(ctx context.Context, in videoIn, emit func(Video) error) error {
		id, err := ParseVideoID(in.Ref)
		if err != nil {
			return errs.Usage("%s", err.Error())
		}
		author := in.Author
		if author == "" {
			if h, herr := ParseHandle(in.Ref); herr == nil {
				author = h
			}
		}
		in.Sess.Progressf("fetching video %s", id)
		v, err := in.Sess.Client.VideoByID(ctx, author, id)
		if err != nil {
			return MapErr(err)
		}
		return emit(v)
	})
}

// --- posts (members of a user) ---

type postsIn struct {
	Ref    string   `kit:"arg" help:"@handle or secUid"`
	Cursor string   `kit:"flag" help:"resume from a paging cursor"`
	Limit  int      `kit:"flag,inherit" help:"max records"`
	Sess   *Session `kit:"inject"`
}

func registerPosts(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "posts", Group: "read", List: true,
		Summary: "A user's public videos",
		URIType: "user",
		Args:    []kit.Arg{{Name: "handle-or-secuid", Help: "@handle or secUid"}},
	}, func(ctx context.Context, in postsIn, emit func(Video) error) error {
		in.Sess.Progressf("fetching posts for %s", in.Ref)
		vids, err := in.Sess.Client.Posts(ctx, in.Ref, effectiveLimit(in.Limit, 35), in.Cursor)
		if err != nil {
			return MapErr(err)
		}
		return emitAll(vids, emit)
	})
}

// --- comments (members of a video) ---

type commentsIn struct {
	Ref     string   `kit:"arg" help:"video url or id"`
	Author  string   `kit:"flag" help:"author handle, used to build the url field"`
	Replies bool     `kit:"flag" help:"expand every thread inline"`
	Limit   int      `kit:"flag,inherit" help:"max records"`
	Sess    *Session `kit:"inject"`
}

func registerComments(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "comments", Group: "read", List: true,
		Summary: "Comments under a video",
		URIType: "video",
		Args:    []kit.Arg{{Name: "url-or-id", Help: "video url or numeric id"}},
	}, func(ctx context.Context, in commentsIn, emit func(Comment) error) error {
		id, err := ParseVideoID(in.Ref)
		if err != nil {
			return errs.Usage("%s", err.Error())
		}
		author := in.Author
		if author == "" {
			if h, herr := ParseHandle(in.Ref); herr == nil {
				author = h
			}
		}
		in.Sess.Progressf("fetching comments for video %s", id)
		comments, err := in.Sess.Client.Comments(ctx, id, author, effectiveLimit(in.Limit, 50))
		if err != nil {
			return MapErr(err)
		}
		for _, c := range comments {
			if err := emit(c); err != nil {
				return err
			}
			if !in.Replies || c.ReplyCount == 0 {
				continue
			}
			in.Sess.Progressf("fetching replies for comment %s", c.ID)
			reps, err := in.Sess.Client.Replies(ctx, id, c.ID, author, 0)
			if err != nil {
				return MapErr(err)
			}
			if err := emitAll(reps, emit); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- replies ---

type repliesIn struct {
	Ref       string   `kit:"arg" help:"video url or id"`
	CommentID string   `kit:"arg" help:"the parent comment id"`
	Author    string   `kit:"flag" help:"author handle, used to build the url field"`
	Limit     int      `kit:"flag,inherit" help:"max records"`
	Sess      *Session `kit:"inject"`
}

func registerReplies(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "replies", Group: "read",
		Summary: "Replies under one comment",
		Args: []kit.Arg{
			{Name: "url-or-id", Help: "video url or numeric id"},
			{Name: "comment-id", Help: "the parent comment id"},
		},
	}, func(ctx context.Context, in repliesIn, emit func(Comment) error) error {
		id, err := ParseVideoID(in.Ref)
		if err != nil {
			return errs.Usage("%s", err.Error())
		}
		author := in.Author
		if author == "" {
			if h, herr := ParseHandle(in.Ref); herr == nil {
				author = h
			}
		}
		in.Sess.Progressf("fetching replies for comment %s", in.CommentID)
		reps, err := in.Sess.Client.Replies(ctx, id, in.CommentID, author, effectiveLimit(in.Limit, 50))
		if err != nil {
			return MapErr(err)
		}
		return emitAll(reps, emit)
	})
}

// --- search ---

type searchIn struct {
	Query []string `kit:"arg,variadic" help:"search terms"`
	Limit int      `kit:"flag,inherit" help:"max records"`
	Sess  *Session `kit:"inject"`
}

func registerSearch(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "search", Group: "read",
		Summary: "Mixed search hits (videos and users)",
		Args:    []kit.Arg{{Name: "query", Help: "search terms", Variadic: true}},
	}, func(ctx context.Context, in searchIn, emit func(SearchHit) error) error {
		q := strings.Join(in.Query, " ")
		in.Sess.Progressf("searching for %q", q)
		hits, err := in.Sess.Client.Search(ctx, q, effectiveLimit(in.Limit, 20))
		if err != nil {
			return MapErr(err)
		}
		return emitAll(hits, emit)
	})
}

// --- users (user search) ---

func registerUsers(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "users", Group: "read",
		Summary: "User search hits",
		URIType: "user",
		Args:    []kit.Arg{{Name: "query", Help: "search terms", Variadic: true}},
	}, func(ctx context.Context, in searchIn, emit func(User) error) error {
		q := strings.Join(in.Query, " ")
		in.Sess.Progressf("searching users for %q", q)
		users, err := in.Sess.Client.Users(ctx, q, effectiveLimit(in.Limit, 20))
		if err != nil {
			return MapErr(err)
		}
		return emitAll(users, emit)
	})
}

// --- hashtag (header, or its videos with --videos) ---

type hashtagIn struct {
	Name   string   `kit:"arg" help:"hashtag name or url"`
	Videos bool     `kit:"flag" help:"emit the hashtag's videos instead of the header record"`
	Limit  int      `kit:"flag,inherit" help:"max records"`
	Sess   *Session `kit:"inject"`
}

func registerHashtag(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "hashtag", Group: "read",
		Summary:  "Hashtag record, or its videos with --videos",
		URIType:  "hashtag",
		Resolver: true,
		Args:     []kit.Arg{{Name: "name", Help: "hashtag name or url"}},
	}, func(ctx context.Context, in hashtagIn, emit func(any) error) error {
		name := in.Name
		if n, ok := tagName(in.Name); ok {
			name = n
		}
		in.Sess.Progressf("fetching hashtag #%s", name)
		tag, err := in.Sess.Client.HashtagByName(ctx, name)
		if err != nil {
			return MapErr(err)
		}
		if !in.Videos {
			return emit(tag)
		}
		in.Sess.Progressf("fetching videos for #%s", name)
		vids, err := in.Sess.Client.HashtagVideos(ctx, tag.ID, effectiveLimit(in.Limit, 30))
		if err != nil {
			return MapErr(err)
		}
		for _, v := range vids {
			if err := emit(v); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- sound (header, or its videos with --videos) ---

type soundIn struct {
	Ref    string   `kit:"arg" help:"sound url or id"`
	Videos bool     `kit:"flag" help:"emit the sound's videos instead of the header record"`
	Limit  int      `kit:"flag,inherit" help:"max records"`
	Sess   *Session `kit:"inject"`
}

func registerSound(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "sound", Group: "read",
		Summary:  "Sound record, or its videos with --videos",
		URIType:  "sound",
		Resolver: true,
		Args:     []kit.Arg{{Name: "url-or-id", Help: "sound url or numeric id"}},
	}, func(ctx context.Context, in soundIn, emit func(any) error) error {
		id, err := ParseMusicID(in.Ref)
		if err != nil {
			return errs.Usage("%s", err.Error())
		}
		in.Sess.Progressf("fetching sound %s", id)
		snd, err := in.Sess.Client.SoundByID(ctx, "", id)
		if err != nil {
			return MapErr(err)
		}
		if !in.Videos {
			return emit(snd)
		}
		in.Sess.Progressf("fetching videos for sound %s", id)
		vids, err := in.Sess.Client.SoundVideos(ctx, id, effectiveLimit(in.Limit, 30))
		if err != nil {
			return MapErr(err)
		}
		for _, v := range vids {
			if err := emit(v); err != nil {
				return err
			}
		}
		return nil
	})
}

// --- trending ---

type trendingIn struct {
	Limit int      `kit:"flag,inherit" help:"max records"`
	Sess  *Session `kit:"inject"`
}

func registerTrending(app *kit.App) {
	kit.Handle(app, kit.OpMeta{
		Name: "trending", Group: "read",
		Summary: "Logged-out recommend feed",
	}, func(ctx context.Context, in trendingIn, emit func(Video) error) error {
		in.Sess.Progressf("fetching trending feed")
		vids, err := in.Sess.Client.Trending(ctx, effectiveLimit(in.Limit, 30))
		if err != nil {
			return MapErr(err)
		}
		return emitAll(vids, emit)
	})
}

// emitAll streams a slice through emit, stopping on the first error (kit's stop
// sentinel once --limit is reached, or a real one).
func emitAll[T any](items []T, emit func(T) error) error {
	for _, it := range items {
		if err := emit(it); err != nil {
			return err
		}
	}
	return nil
}
