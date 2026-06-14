package tiktok

// Mappers from raw wire structs to the clean records. They centralize the
// counter fallbacks and the url building so the SSR and API paths agree.

func userFrom(u rawUser, s rawStats) User {
	avatar := u.AvatarLarger
	if avatar == "" {
		avatar = u.AvatarMedium
	}
	heart := s.HeartCount
	if heart == 0 {
		heart = s.Heart
	}
	return User{
		ID:             u.ID,
		UniqueID:       u.UniqueID,
		Nickname:       u.Nickname,
		SecUID:         u.SecUID,
		Signature:      u.Signature,
		Verified:       u.Verified,
		Private:        u.PrivateAccount,
		Region:         u.Region,
		FollowerCount:  int64(s.FollowerCount),
		FollowingCount: int64(s.FollowingCount),
		HeartCount:     int64(heart),
		VideoCount:     int64(s.VideoCount),
		FriendCount:    int64(s.FriendCount),
		Avatar:         avatar,
		URL:            Host + "/@" + u.UniqueID,
	}
}

func videoFrom(it rawItem) Video {
	challenges := make([]string, 0, len(it.Challenges))
	for _, c := range it.Challenges {
		if c.Title != "" {
			challenges = append(challenges, c.Title)
		}
	}
	// Prefer the numeric stats, fall back to the string typed statsV2.
	digg := pick(it.Stats.DiggCount, it.StatsV2.DiggCount)
	share := pick(it.Stats.ShareCount, it.StatsV2.ShareCount)
	comment := pick(it.Stats.CommentCount, it.StatsV2.CommentCount)
	play := pick(it.Stats.PlayCount, it.StatsV2.PlayCount)
	collect := pick(it.Stats.CollectCount, it.StatsV2.CollectCount)

	cover := it.Video.Cover
	if cover == "" {
		cover = it.Video.OriginCover
	}
	return Video{
		ID:           it.ID,
		Desc:         it.Desc,
		CreateTime:   int64(it.CreateTime),
		Author:       it.Author.UniqueID,
		AuthorID:     it.Author.ID,
		AuthorSecUID: it.Author.SecUID,
		MusicID:      it.Music.ID,
		MusicTitle:   it.Music.Title,
		MusicAuthor:  it.Music.AuthorName,
		Challenges:   challenges,
		Duration:     int64(it.Video.Duration),
		Cover:        cover,
		PlayAddr:     it.Video.PlayAddr,
		DownloadAddr: it.Video.DownloadAddr,
		Width:        int64(it.Video.Width),
		Height:       int64(it.Video.Height),
		DiggCount:    digg,
		ShareCount:   share,
		CommentCount: comment,
		PlayCount:    play,
		CollectCount: collect,
		URL:          videoURL(it.Author.UniqueID, it.ID),
	}
}

func commentFrom(c rawComment, author string) Comment {
	return Comment{
		ID:         c.CID,
		VideoID:    c.AwemeID,
		Text:       c.Text,
		Author:     c.User.UniqueID,
		AuthorID:   c.User.UID,
		AuthorNick: c.User.Nickname,
		CreateTime: int64(c.CreateTime),
		DiggCount:  int64(c.DiggCount),
		ReplyCount: int64(c.ReplyTotal),
		ParentID:   "",
		URL:        videoURL(author, c.AwemeID),
	}
}

func soundFrom(m rawMusic, videoCount int64) Sound {
	return Sound{
		ID:         m.ID,
		Title:      m.Title,
		AuthorName: m.AuthorName,
		Original:   m.Original,
		Duration:   int64(m.Duration),
		PlayURL:    m.PlayURL,
		Cover:      m.CoverLarge,
		VideoCount: videoCount,
		URL:        Host + "/music/-" + m.ID,
	}
}

func hashtagFrom(c rawChallenge, videoCount, viewCount int64) Hashtag {
	return Hashtag{
		ID:         c.ID,
		Title:      c.Title,
		Desc:       c.Desc,
		VideoCount: videoCount,
		ViewCount:  viewCount,
		Cover:      c.CoverLarger,
		URL:        Host + "/tag/" + c.Title,
	}
}

// pick returns the first non-zero counter.
func pick(a, b flexInt) int64 {
	if a != 0 {
		return int64(a)
	}
	return int64(b)
}
