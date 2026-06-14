package tiktok

import (
	"bytes"
	"encoding/json"
	"strconv"
)

// flexInt decodes a counter that TikTok sends as a JSON number in `stats` and
// as a quoted string in `statsV2`. Both shapes land here as int64.
type flexInt int64

func (f *flexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s == "" {
			*f = 0
			return nil
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			*f = 0
			return nil
		}
		*f = flexInt(n)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	v, err := n.Int64()
	if err != nil {
		*f = 0
		return nil
	}
	*f = flexInt(v)
	return nil
}

// The raw wire shapes. They mirror the parts of the TikTok JSON the tool reads
// and ignore the rest.

type rawUniversal struct {
	DefaultScope struct {
		UserDetail      rawUserDetail      `json:"webapp.user-detail"`
		VideoDetail     rawVideoDetail     `json:"webapp.video-detail"`
		ChallengeDetail rawChallengeDetail `json:"webapp.challenge-detail"`
		MusicDetail     rawMusicDetail     `json:"webapp.music-detail"`
		AppContext      rawAppContext      `json:"webapp.app-context"`
	} `json:"__DEFAULT_SCOPE__"`
}

type rawAppContext struct {
	WID       string `json:"wid"`
	Region    string `json:"region"`
	CSRFToken string `json:"csrfToken"`
}

type rawUserDetail struct {
	UserInfo rawUserInfo `json:"userInfo"`
}

type rawUserInfo struct {
	User  rawUser  `json:"user"`
	Stats rawStats `json:"stats"`
}

type rawUser struct {
	ID             string `json:"id"`
	UniqueID       string `json:"uniqueId"`
	Nickname       string `json:"nickname"`
	SecUID         string `json:"secUid"`
	Signature      string `json:"signature"`
	Verified       bool   `json:"verified"`
	PrivateAccount bool   `json:"privateAccount"`
	Region         string `json:"region"`
	AvatarLarger   string `json:"avatarLarger"`
	AvatarMedium   string `json:"avatarMedium"`
}

type rawStats struct {
	FollowerCount  flexInt `json:"followerCount"`
	FollowingCount flexInt `json:"followingCount"`
	HeartCount     flexInt `json:"heartCount"`
	Heart          flexInt `json:"heart"`
	VideoCount     flexInt `json:"videoCount"`
	FriendCount    flexInt `json:"friendCount"`
	DiggCount      flexInt `json:"diggCount"`
	ShareCount     flexInt `json:"shareCount"`
	CommentCount   flexInt `json:"commentCount"`
	PlayCount      flexInt `json:"playCount"`
	CollectCount   flexInt `json:"collectCount"`
}

type rawVideoDetail struct {
	ItemInfo struct {
		ItemStruct rawItem `json:"itemStruct"`
	} `json:"itemInfo"`
}

type rawItem struct {
	ID         string         `json:"id"`
	Desc       string         `json:"desc"`
	CreateTime flexInt        `json:"createTime"`
	Author     rawUser        `json:"author"`
	Music      rawMusic       `json:"music"`
	Challenges []rawChallenge `json:"challenges"`
	Video      rawVideoMedia  `json:"video"`
	Stats      rawStats       `json:"stats"`
	StatsV2    rawStatsV2     `json:"statsV2"`
}

type rawStatsV2 struct {
	DiggCount    flexInt `json:"diggCount"`
	ShareCount   flexInt `json:"shareCount"`
	CommentCount flexInt `json:"commentCount"`
	PlayCount    flexInt `json:"playCount"`
	CollectCount flexInt `json:"collectCount"`
}

type rawVideoMedia struct {
	Duration     flexInt `json:"duration"`
	Cover        string  `json:"cover"`
	OriginCover  string  `json:"originCover"`
	PlayAddr     string  `json:"playAddr"`
	DownloadAddr string  `json:"downloadAddr"`
	Width        flexInt `json:"width"`
	Height       flexInt `json:"height"`
}

type rawMusic struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	AuthorName string  `json:"authorName"`
	Original   bool    `json:"original"`
	Duration   flexInt `json:"duration"`
	PlayURL    string  `json:"playUrl"`
	CoverLarge string  `json:"coverLarge"`
	UserCount  flexInt `json:"userCount"`
}

type rawChallenge struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Desc        string `json:"desc"`
	CoverLarger string `json:"coverLarger"`
	Stats       struct {
		VideoCount flexInt `json:"videoCount"`
		ViewCount  flexInt `json:"viewCount"`
	} `json:"stats"`
}

type rawChallengeDetail struct {
	ChallengeInfo struct {
		Challenge rawChallenge `json:"challenge"`
		Stats     struct {
			VideoCount flexInt `json:"videoCount"`
			ViewCount  flexInt `json:"viewCount"`
		} `json:"stats"`
	} `json:"challengeInfo"`
}

type rawMusicDetail struct {
	MusicInfo struct {
		Music rawMusic `json:"music"`
		Stats struct {
			VideoCount flexInt `json:"videoCount"`
		} `json:"stats"`
	} `json:"musicInfo"`
}

// API list envelopes.

type rawItemList struct {
	StatusCode flexInt   `json:"statusCode"`
	ItemList   []rawItem `json:"itemList"`
	HasMore    bool      `json:"hasMore"`
	Cursor     flexInt   `json:"cursor"`
}

type rawCommentList struct {
	StatusCode flexInt      `json:"status_code"`
	Comments   []rawComment `json:"comments"`
	HasMore    flexInt      `json:"has_more"`
	Cursor     flexInt      `json:"cursor"`
	Total      flexInt      `json:"total"`
}

type rawSearch struct {
	StatusCode flexInt `json:"status_code"`
	Data       []struct {
		Type     flexInt `json:"type"`
		Item     rawItem `json:"item"`
		UserInfo struct {
			User rawUser `json:"user"`
		} `json:"user_info"`
	} `json:"data"`
	HasMore  flexInt `json:"has_more"`
	Cursor   flexInt `json:"cursor"`
	SearchID string  `json:"log_pb"`
}

type rawUserSearch struct {
	StatusCode flexInt `json:"status_code"`
	UserList   []struct {
		UserInfo struct {
			User  rawUser  `json:"user"`
			Stats rawStats `json:"stats"`
		} `json:"user_info"`
	} `json:"user_list"`
	HasMore flexInt `json:"has_more"`
	Cursor  flexInt `json:"cursor"`
}

type rawComment struct {
	CID        string  `json:"cid"`
	AwemeID    string  `json:"aweme_id"`
	Text       string  `json:"text"`
	CreateTime flexInt `json:"create_time"`
	DiggCount  flexInt `json:"digg_count"`
	ReplyTotal flexInt `json:"reply_comment_total"`
	User       struct {
		UID      string `json:"uid"`
		UniqueID string `json:"unique_id"`
		Nickname string `json:"nickname"`
		SecUID   string `json:"sec_uid"`
	} `json:"user"`
}
