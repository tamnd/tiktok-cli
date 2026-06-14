package cli

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/tamnd/tiktok-cli/tiktok"
)

func (a *App) postsCmd() *cobra.Command {
	var cursor string
	cmd := &cobra.Command{
		Use:   "posts <handle|secUid>",
		Short: "A user's public videos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching posts for %s", args[0])
			vids, err := a.client.Posts(cmd.Context(), args[0], a.effectiveLimit(35), cursor)
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(vids, len(vids))
		},
	}
	cmd.Flags().StringVar(&cursor, "cursor", "", "resume from a paging cursor")
	return cmd
}

func (a *App) commentsCmd() *cobra.Command {
	var author string
	var replies bool
	cmd := &cobra.Command{
		Use:   "comments <url|id>",
		Short: "Comments under a video",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := tiktok.ParseVideoID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			if author == "" {
				if h, herr := tiktok.ParseHandle(args[0]); herr == nil {
					author = h
				}
			}
			a.progressf("fetching comments for video %s", id)
			comments, err := a.client.Comments(cmd.Context(), id, author, a.effectiveLimit(50))
			if err != nil {
				return mapErr(err)
			}
			if replies {
				comments, err = a.expandReplies(cmd, id, author, comments)
				if err != nil {
					return mapErr(err)
				}
			}
			return a.renderList(comments, len(comments))
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "author handle, used to build the url field")
	cmd.Flags().BoolVar(&replies, "replies", false, "expand every thread inline")
	return cmd
}

// expandReplies walks each top-level comment and appends its replies inline.
func (a *App) expandReplies(cmd *cobra.Command, videoID, author string, top []tiktok.Comment) ([]tiktok.Comment, error) {
	out := make([]tiktok.Comment, 0, len(top))
	for _, c := range top {
		out = append(out, c)
		if c.ReplyCount == 0 {
			continue
		}
		a.progressf("fetching replies for comment %s", c.ID)
		reps, err := a.client.Replies(cmd.Context(), videoID, c.ID, author, 0)
		if err != nil {
			return out, err
		}
		out = append(out, reps...)
	}
	return out, nil
}

func (a *App) repliesCmd() *cobra.Command {
	var author string
	cmd := &cobra.Command{
		Use:   "replies <url|id> <comment-id>",
		Short: "Replies under one comment",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := tiktok.ParseVideoID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			if author == "" {
				if h, herr := tiktok.ParseHandle(args[0]); herr == nil {
					author = h
				}
			}
			a.progressf("fetching replies for comment %s", args[1])
			reps, err := a.client.Replies(cmd.Context(), id, args[1], author, a.effectiveLimit(50))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(reps, len(reps))
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "author handle, used to build the url field")
	return cmd
}

func (a *App) searchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Mixed search hits (videos and users)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := joinArgs(args)
			a.progressf("searching for %q", q)
			hits, err := a.client.Search(cmd.Context(), q, a.effectiveLimit(20))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(hits, len(hits))
		},
	}
}

func (a *App) usersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "users <query>",
		Short: "User search hits",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			q := joinArgs(args)
			a.progressf("searching users for %q", q)
			users, err := a.client.Users(cmd.Context(), q, a.effectiveLimit(20))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(users, len(users))
		},
	}
}

func (a *App) trendingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trending",
		Short: "Logged-out recommend feed",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a.progressf("fetching trending feed")
			vids, err := a.client.Trending(cmd.Context(), a.effectiveLimit(30))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(vids, len(vids))
		},
	}
}

func joinArgs(args []string) string {
	return strings.Join(args, " ")
}
