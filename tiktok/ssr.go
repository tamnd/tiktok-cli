package tiktok

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tamnd/tiktok-cli/pkg/tthtml"
)

// parseUniversal extracts and decodes the rehydration blob from a page body.
func parseUniversal(html string) (*rawUniversal, error) {
	raw, err := tthtml.ScriptJSON(html, tthtml.UniversalDataID)
	if err != nil {
		return nil, fmt.Errorf("read universal data: %w", err)
	}
	var u rawUniversal
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, fmt.Errorf("decode universal data: %w", err)
	}
	return &u, nil
}

// UserByHandle fetches a profile page and returns its User record.
func (c *Client) UserByHandle(ctx context.Context, handle string) (User, error) {
	html, err := c.GetPage(ctx, Host+"/@"+handle)
	if err != nil {
		return User{}, err
	}
	u, err := parseUniversal(html)
	if err != nil {
		return User{}, err
	}
	info := u.DefaultScope.UserDetail.UserInfo
	if info.User.UniqueID == "" {
		return User{}, ErrNotFound
	}
	return userFrom(info.User, info.Stats), nil
}

// VideoByID fetches a video page and returns its Video record. The author hint
// builds the canonical url when one is known; when empty the page supplies it.
func (c *Client) VideoByID(ctx context.Context, author, id string) (Video, error) {
	page := Host + "/@" + author + "/video/" + id
	if author == "" {
		page = Host + "/embed/v2/" + id
	}
	html, err := c.GetPage(ctx, page)
	if err != nil {
		return Video{}, err
	}
	u, err := parseUniversal(html)
	if err != nil {
		return Video{}, err
	}
	it := u.DefaultScope.VideoDetail.ItemInfo.ItemStruct
	if it.ID == "" {
		return Video{}, ErrNotFound
	}
	return videoFrom(it), nil
}

// HashtagByName fetches a tag page and returns its Hashtag record.
func (c *Client) HashtagByName(ctx context.Context, name string) (Hashtag, error) {
	html, err := c.GetPage(ctx, Host+"/tag/"+name)
	if err != nil {
		return Hashtag{}, err
	}
	u, err := parseUniversal(html)
	if err != nil {
		return Hashtag{}, err
	}
	ci := u.DefaultScope.ChallengeDetail.ChallengeInfo
	if ci.Challenge.ID == "" && ci.Challenge.Title == "" {
		return Hashtag{}, ErrNotFound
	}
	return hashtagFrom(ci.Challenge, int64(ci.Stats.VideoCount), int64(ci.Stats.ViewCount)), nil
}

// SoundByID fetches a music page and returns its Sound record.
func (c *Client) SoundByID(ctx context.Context, slug, id string) (Sound, error) {
	if slug == "" {
		slug = "x"
	}
	html, err := c.GetPage(ctx, Host+"/music/"+slug+"-"+id)
	if err != nil {
		return Sound{}, err
	}
	u, err := parseUniversal(html)
	if err != nil {
		return Sound{}, err
	}
	mi := u.DefaultScope.MusicDetail.MusicInfo
	if mi.Music.ID == "" {
		return Sound{}, ErrNotFound
	}
	return soundFrom(mi.Music, int64(mi.Stats.VideoCount)), nil
}

// RawUniversal fetches a page and returns its rehydration blob as pretty JSON.
func (c *Client) RawUniversal(ctx context.Context, url string) ([]byte, error) {
	html, err := c.GetPage(ctx, url)
	if err != nil {
		return nil, err
	}
	raw, err := tthtml.ScriptJSON(html, tthtml.UniversalDataID)
	if err != nil {
		return nil, err
	}
	var pretty json.RawMessage = []byte(raw)
	return json.MarshalIndent(pretty, "", "  ")
}

// secUIDForHandle resolves a handle to its secUid through the profile page.
func (c *Client) secUIDForHandle(ctx context.Context, handle string) (string, error) {
	u, err := c.UserByHandle(ctx, handle)
	if err != nil {
		return "", err
	}
	if u.SecUID == "" {
		return "", ErrNotFound
	}
	return u.SecUID, nil
}
