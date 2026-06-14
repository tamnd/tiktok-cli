package tiktok

import (
	"errors"
	"regexp"
	"strings"
)

var (
	reVideoID = regexp.MustCompile(`/video/(\d+)`)
	rePhotoID = regexp.MustCompile(`/photo/(\d+)`)
	reMusicID = regexp.MustCompile(`/music/[^/?]*-(\d+)`)
	reDigits  = regexp.MustCompile(`^\d+$`)
	reHandle  = regexp.MustCompile(`@([A-Za-z0-9_.]+)`)
	reTagURL  = regexp.MustCompile(`/tag/([^/?#]+)`)
)

// tagName recognizes a hashtag reference: a leading '#', or a /tag/{name} url.
// A bare word is intentionally not a hashtag, so a host reads it as a @handle.
func tagName(in string) (string, bool) {
	in = strings.TrimSpace(in)
	if name, ok := strings.CutPrefix(in, "#"); ok && name != "" {
		return name, true
	}
	if m := reTagURL.FindStringSubmatch(in); m != nil {
		return m[1], true
	}
	return "", false
}

// ParseHandle pulls a bare @handle out of "tiktok", "@tiktok", or a full
// profile url.
func ParseHandle(in string) (string, error) {
	in = strings.TrimSpace(in)
	if in == "" {
		return "", errors.New("empty handle")
	}
	if m := reHandle.FindStringSubmatch(in); m != nil {
		return m[1], nil
	}
	in = strings.TrimPrefix(in, "@")
	if strings.ContainsAny(in, "/ ") {
		return "", errors.New("could not read a handle from " + in)
	}
	return in, nil
}

// ParseVideoID pulls a numeric video id out of a /video/{id} or /photo/{id}
// url, or accepts a bare numeric id.
func ParseVideoID(in string) (string, error) {
	in = strings.TrimSpace(in)
	if reDigits.MatchString(in) {
		return in, nil
	}
	if m := reVideoID.FindStringSubmatch(in); m != nil {
		return m[1], nil
	}
	if m := rePhotoID.FindStringSubmatch(in); m != nil {
		return m[1], nil
	}
	return "", errors.New("could not read a video id from " + in)
}

// ParseMusicID pulls a numeric sound id out of a /music/{slug}-{id} url, or
// accepts a bare numeric id.
func ParseMusicID(in string) (string, error) {
	in = strings.TrimSpace(in)
	if reDigits.MatchString(in) {
		return in, nil
	}
	if m := reMusicID.FindStringSubmatch(in); m != nil {
		return m[1], nil
	}
	return "", errors.New("could not read a sound id from " + in)
}

// IsSecUID reports whether s looks like a TikTok secUid rather than a handle.
// A secUid is a long opaque token starting with "MS4wLjAB".
func IsSecUID(s string) bool {
	return strings.HasPrefix(s, "MS4wLjAB")
}

// videoURL builds the canonical url for a video given its author and id.
func videoURL(author, id string) string {
	if author == "" {
		return Host + "/@/video/" + id
	}
	return Host + "/@" + author + "/video/" + id
}
