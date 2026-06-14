package tiktok

// The clean records. Each is a flat struct whose json tags set the default
// column order and whose url field feeds `-o url`.

// User is a public profile.
type User struct {
	ID             string `json:"id"`
	UniqueID       string `json:"unique_id"`
	Nickname       string `json:"nickname"`
	SecUID         string `json:"sec_uid"`
	Signature      string `json:"signature"`
	Verified       bool   `json:"verified"`
	Private        bool   `json:"private"`
	Region         string `json:"region"`
	FollowerCount  int64  `json:"follower_count"`
	FollowingCount int64  `json:"following_count"`
	HeartCount     int64  `json:"heart_count"`
	VideoCount     int64  `json:"video_count"`
	FriendCount    int64  `json:"friend_count"`
	Avatar         string `json:"avatar"`
	URL            string `json:"url"`
}

// Video is a single post with its author, sound, hashtags, and counters.
type Video struct {
	ID           string   `json:"id"`
	Desc         string   `json:"desc"`
	CreateTime   int64    `json:"create_time"`
	Author       string   `json:"author"`
	AuthorID     string   `json:"author_id"`
	AuthorSecUID string   `json:"author_sec_uid"`
	MusicID      string   `json:"music_id"`
	MusicTitle   string   `json:"music_title"`
	MusicAuthor  string   `json:"music_author"`
	Challenges   []string `json:"challenges"`
	Duration     int64    `json:"duration"`
	Cover        string   `json:"cover"`
	PlayAddr     string   `json:"play_addr"`
	DownloadAddr string   `json:"download_addr"`
	Width        int64    `json:"width"`
	Height       int64    `json:"height"`
	DiggCount    int64    `json:"digg_count"`
	ShareCount   int64    `json:"share_count"`
	CommentCount int64    `json:"comment_count"`
	PlayCount    int64    `json:"play_count"`
	CollectCount int64    `json:"collect_count"`
	URL          string   `json:"url"`
}

// Comment is one comment under a video, with its parent id for replies.
type Comment struct {
	ID         string `json:"id"`
	VideoID    string `json:"video_id"`
	Text       string `json:"text"`
	Author     string `json:"author"`
	AuthorID   string `json:"author_id"`
	AuthorNick string `json:"author_nick"`
	CreateTime int64  `json:"create_time"`
	DiggCount  int64  `json:"digg_count"`
	ReplyCount int64  `json:"reply_count"`
	ParentID   string `json:"parent_id"`
	URL        string `json:"url"`
}

// Hashtag is a challenge page header.
type Hashtag struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Desc       string `json:"desc"`
	VideoCount int64  `json:"video_count"`
	ViewCount  int64  `json:"view_count"`
	Cover      string `json:"cover"`
	URL        string `json:"url"`
}

// Sound is a music page header.
type Sound struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	AuthorName string `json:"author_name"`
	Original   bool   `json:"original"`
	Duration   int64  `json:"duration"`
	PlayURL    string `json:"play_url"`
	Cover      string `json:"cover"`
	VideoCount int64  `json:"video_count"`
	URL        string `json:"url"`
}

// SearchHit is a thin, normalized search result row.
type SearchHit struct {
	Type   string `json:"type"`
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
	URL    string `json:"url"`
}
