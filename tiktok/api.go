package tiktok

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
)

// commonParams is the device and browser block every signed call carries.
func commonParams() url.Values {
	v := url.Values{}
	v.Set("aid", "1988")
	v.Set("app_name", "tiktok_web")
	v.Set("device_platform", "web_pc")
	v.Set("region", "US")
	v.Set("language", "en")
	v.Set("cookie_enabled", "true")
	v.Set("screen_width", "1920")
	v.Set("screen_height", "1080")
	v.Set("browser_language", "en-US")
	v.Set("browser_platform", "MacIntel")
	v.Set("browser_name", "Mozilla")
	v.Set("browser_online", "true")
	v.Set("os", "mac")
	v.Set("channel", "tiktok_web")
	v.Set("webcast_language", "en")
	return v
}

// Posts pages a user's videos through post/item_list. handle may be a @handle
// or a secUid. It returns up to limit videos (0 means one page).
func (c *Client) Posts(ctx context.Context, handle string, limit int, cursor string) ([]Video, error) {
	secUID := handle
	if !IsSecUID(handle) {
		h, err := ParseHandle(handle)
		if err != nil {
			return nil, err
		}
		secUID, err = c.secUIDForHandle(ctx, h)
		if err != nil {
			return nil, err
		}
	}

	var out []Video
	cur := cursor
	if cur == "" {
		cur = "0"
	}
	for {
		v := commonParams()
		v.Set("secUid", secUID)
		v.Set("count", "35")
		v.Set("cursor", cur)
		body, err := c.GetAPI(ctx, "/api/post/item_list/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawItemList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, it := range list.ItemList {
			out = append(out, videoFrom(it))
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
		if !list.HasMore || len(list.ItemList) == 0 {
			break
		}
		cur = strconv.FormatInt(int64(list.Cursor), 10)
	}
	return out, nil
}

// Comments pages the comment list under a video. author builds the url field.
func (c *Client) Comments(ctx context.Context, videoID, author string, limit int) ([]Comment, error) {
	var out []Comment
	cur := int64(0)
	for {
		v := commonParams()
		v.Set("aweme_id", videoID)
		v.Set("count", "50")
		v.Set("cursor", strconv.FormatInt(cur, 10))
		body, err := c.GetAPI(ctx, "/api/comment/list/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawCommentList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, rc := range list.Comments {
			out = append(out, commentFrom(rc, author))
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
		if list.HasMore == 0 || len(list.Comments) == 0 {
			break
		}
		cur = int64(list.Cursor)
	}
	return out, nil
}

// Replies pages the replies under one comment.
func (c *Client) Replies(ctx context.Context, videoID, commentID, author string, limit int) ([]Comment, error) {
	var out []Comment
	cur := int64(0)
	for {
		v := commonParams()
		v.Set("item_id", videoID)
		v.Set("comment_id", commentID)
		v.Set("count", "50")
		v.Set("cursor", strconv.FormatInt(cur, 10))
		body, err := c.GetAPI(ctx, "/api/comment/list/reply/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawCommentList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, rc := range list.Comments {
			cm := commentFrom(rc, author)
			cm.ParentID = commentID
			out = append(out, cm)
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
		if list.HasMore == 0 || len(list.Comments) == 0 {
			break
		}
		cur = int64(list.Cursor)
	}
	return out, nil
}

// HashtagVideos pages the videos under a challenge id.
func (c *Client) HashtagVideos(ctx context.Context, challengeID string, limit int) ([]Video, error) {
	var out []Video
	cur := int64(0)
	for {
		v := commonParams()
		v.Set("challengeID", challengeID)
		v.Set("count", "30")
		v.Set("cursor", strconv.FormatInt(cur, 10))
		body, err := c.GetAPI(ctx, "/api/challenge/item_list/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawItemList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, it := range list.ItemList {
			out = append(out, videoFrom(it))
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
		if !list.HasMore || len(list.ItemList) == 0 {
			break
		}
		cur = int64(list.Cursor)
	}
	return out, nil
}

// SoundVideos pages the videos using a sound id.
func (c *Client) SoundVideos(ctx context.Context, musicID string, limit int) ([]Video, error) {
	var out []Video
	cur := int64(0)
	for {
		v := commonParams()
		v.Set("musicID", musicID)
		v.Set("count", "30")
		v.Set("cursor", strconv.FormatInt(cur, 10))
		body, err := c.GetAPI(ctx, "/api/music/item_list/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawItemList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, it := range list.ItemList {
			out = append(out, videoFrom(it))
			if limit > 0 && len(out) >= limit {
				return out[:limit], nil
			}
		}
		if !list.HasMore || len(list.ItemList) == 0 {
			break
		}
		cur = int64(list.Cursor)
	}
	return out, nil
}

// Search runs a mixed search and returns thin, normalized hits.
func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]SearchHit, error) {
	if limit <= 0 {
		limit = 20
	}
	v := commonParams()
	v.Set("keyword", keyword)
	v.Set("offset", "0")
	v.Set("count", strconv.Itoa(limit))
	body, err := c.GetAPI(ctx, "/api/search/general/full/", v.Encode())
	if err != nil {
		return nil, err
	}
	var res rawSearch
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	out := make([]SearchHit, 0, len(res.Data))
	for _, d := range res.Data {
		switch {
		case d.Item.ID != "":
			out = append(out, SearchHit{
				Type:   "video",
				ID:     d.Item.ID,
				Title:  d.Item.Desc,
				Author: d.Item.Author.UniqueID,
				URL:    videoURL(d.Item.Author.UniqueID, d.Item.ID),
			})
		case d.UserInfo.User.UniqueID != "":
			u := d.UserInfo.User
			out = append(out, SearchHit{
				Type:  "user",
				ID:    u.ID,
				Title: u.Nickname,
				URL:   Host + "/@" + u.UniqueID,
			})
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Users runs a user search and returns the matched profiles.
func (c *Client) Users(ctx context.Context, keyword string, limit int) ([]User, error) {
	if limit <= 0 {
		limit = 20
	}
	v := commonParams()
	v.Set("keyword", keyword)
	v.Set("offset", "0")
	v.Set("count", strconv.Itoa(limit))
	body, err := c.GetAPI(ctx, "/api/search/user/full/", v.Encode())
	if err != nil {
		return nil, err
	}
	var res rawUserSearch
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	out := make([]User, 0, len(res.UserList))
	for _, u := range res.UserList {
		out = append(out, userFrom(u.UserInfo.User, u.UserInfo.Stats))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

// Trending pages the logged-out recommend feed.
func (c *Client) Trending(ctx context.Context, limit int) ([]Video, error) {
	if limit <= 0 {
		limit = 30
	}
	v := commonParams()
	v.Set("count", strconv.Itoa(limit))
	v.Set("pullType", "1")
	body, err := c.GetAPI(ctx, "/api/recommend/item_list/", v.Encode())
	if err != nil {
		return nil, err
	}
	var list rawItemList
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, err
	}
	out := make([]Video, 0, len(list.ItemList))
	for _, it := range list.ItemList {
		out = append(out, videoFrom(it))
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
