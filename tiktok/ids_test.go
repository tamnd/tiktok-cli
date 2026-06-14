package tiktok

import "testing"

func TestParseHandle(t *testing.T) {
	cases := map[string]string{
		"tiktok":                           "tiktok",
		"@tiktok":                          "tiktok",
		"https://www.tiktok.com/@tiktok":   "tiktok",
		"https://www.tiktok.com/@charli.d": "charli.d",
	}
	for in, want := range cases {
		got, err := ParseHandle(in)
		if err != nil {
			t.Errorf("ParseHandle(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseHandle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseVideoID(t *testing.T) {
	cases := map[string]string{
		"7106594312292453675": "7106594312292453675",
		"https://www.tiktok.com/@tiktok/video/7106594312292453675": "7106594312292453675",
		"https://www.tiktok.com/@user/photo/7200000000000000000":   "7200000000000000000",
	}
	for in, want := range cases {
		got, err := ParseVideoID(in)
		if err != nil {
			t.Errorf("ParseVideoID(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseVideoID(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := ParseVideoID("not-a-video"); err == nil {
		t.Error("expected error for a non-video input")
	}
}

func TestParseMusicID(t *testing.T) {
	got, err := ParseMusicID("https://www.tiktok.com/music/original-sound-7106594300000000000")
	if err != nil {
		t.Fatal(err)
	}
	if got != "7106594300000000000" {
		t.Fatalf("got %q", got)
	}
}

func TestIsSecUID(t *testing.T) {
	if !IsSecUID("MS4wLjABAAAAv7iSuuXyz") {
		t.Error("expected secUid to be recognized")
	}
	if IsSecUID("tiktok") {
		t.Error("a handle is not a secUid")
	}
}
