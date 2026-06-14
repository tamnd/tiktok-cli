package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/tiktok-cli/tiktok"
)

func (a *App) userCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "user <handle>",
		Short: "Profile record for a @handle",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			handle, err := tiktok.ParseHandle(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			a.progressf("fetching profile @%s", handle)
			u, err := a.client.UserByHandle(cmd.Context(), handle)
			if err != nil {
				return mapErr(err)
			}
			return a.render(u)
		},
	}
}

func (a *App) videoCmd() *cobra.Command {
	var author string
	cmd := &cobra.Command{
		Use:   "video <url|id>",
		Short: "One video record",
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
			a.progressf("fetching video %s", id)
			v, err := a.client.VideoByID(cmd.Context(), author, id)
			if err != nil {
				return mapErr(err)
			}
			return a.render(v)
		},
	}
	cmd.Flags().StringVar(&author, "author", "", "author handle, used to build the canonical url for a bare id")
	return cmd
}

func (a *App) hashtagCmd() *cobra.Command {
	var videos bool
	cmd := &cobra.Command{
		Use:   "hashtag <name>",
		Short: "Hashtag record, or its videos with --videos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			a.progressf("fetching hashtag #%s", name)
			tag, err := a.client.HashtagByName(cmd.Context(), name)
			if err != nil {
				return mapErr(err)
			}
			if !videos {
				return a.render(tag)
			}
			a.progressf("fetching videos for #%s", name)
			vids, err := a.client.HashtagVideos(cmd.Context(), tag.ID, a.effectiveLimit(30))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(vids, len(vids))
		},
	}
	cmd.Flags().BoolVar(&videos, "videos", false, "emit the hashtag's videos instead of the header record")
	return cmd
}

func (a *App) soundCmd() *cobra.Command {
	var videos bool
	cmd := &cobra.Command{
		Use:   "sound <url|id>",
		Short: "Sound record, or its videos with --videos",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := tiktok.ParseMusicID(args[0])
			if err != nil {
				return codeError(exitUsage, err)
			}
			a.progressf("fetching sound %s", id)
			snd, err := a.client.SoundByID(cmd.Context(), "", id)
			if err != nil {
				return mapErr(err)
			}
			if !videos {
				return a.render(snd)
			}
			a.progressf("fetching videos for sound %s", id)
			vids, err := a.client.SoundVideos(cmd.Context(), id, a.effectiveLimit(30))
			if err != nil {
				return mapErr(err)
			}
			return a.renderList(vids, len(vids))
		},
	}
	cmd.Flags().BoolVar(&videos, "videos", false, "emit the sound's videos instead of the header record")
	return cmd
}

func (a *App) rawCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "raw <url>",
		Short: "Print a page's universal-data blob as pretty JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a.progressf("fetching %s", args[0])
			out, err := a.client.RawUniversal(cmd.Context(), args[0])
			if err != nil {
				return mapErr(err)
			}
			cmd.Println(string(out))
			return nil
		},
	}
}
