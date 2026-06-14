package tiktok

import (
	"context"
	"encoding/json"
	"strconv"
)

// Related pages the videos TikTok recommends alongside one video, through
// related/item_list. It returns up to limit videos (0 means one page). The
// discovery walk uses it as a video-to-video edge.
func (c *Client) Related(ctx context.Context, videoID string, limit int) ([]Video, error) {
	if limit <= 0 {
		limit = 16
	}
	var out []Video
	cur := int64(0)
	for {
		v := commonParams()
		v.Set("itemID", videoID)
		v.Set("count", "16")
		v.Set("cursor", strconv.FormatInt(cur, 10))
		body, err := c.GetAPI(ctx, "/api/related/item_list/", v.Encode())
		if err != nil {
			return out, err
		}
		var list rawItemList
		if err := json.Unmarshal(body, &list); err != nil {
			return out, err
		}
		for _, it := range list.ItemList {
			out = append(out, videoFrom(it))
			if len(out) >= limit {
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
