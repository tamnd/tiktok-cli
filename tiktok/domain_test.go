package tiktok

import "testing"

func TestClassify(t *testing.T) {
	cases := []struct {
		in       string
		wantType string
		wantID   string
	}{
		{"tiktok", "user", "tiktok"},
		{"@tiktok", "user", "tiktok"},
		{"https://www.tiktok.com/@tiktok", "user", "tiktok"},
		{"7212345678901234567", "video", "7212345678901234567"},
		{"https://www.tiktok.com/@scout2015/video/7212345678901234567", "video", "7212345678901234567"},
		{"#fyp", "hashtag", "fyp"},
		{"https://www.tiktok.com/tag/fyp", "hashtag", "fyp"},
		{"https://www.tiktok.com/music/original-sound-6889283426862778117", "sound", "6889283426862778117"},
	}
	for _, c := range cases {
		gotType, gotID, err := Domain{}.Classify(c.in)
		if err != nil {
			t.Errorf("Classify(%q) error: %v", c.in, err)
			continue
		}
		if gotType != c.wantType || gotID != c.wantID {
			t.Errorf("Classify(%q) = (%q, %q), want (%q, %q)", c.in, gotType, gotID, c.wantType, c.wantID)
		}
	}
}

func TestClassifyUnrecognized(t *testing.T) {
	if _, _, err := (Domain{}).Classify(""); err == nil {
		t.Fatal("Classify(\"\") want error, got nil")
	}
}

func TestLocate(t *testing.T) {
	cases := []struct {
		uriType string
		id      string
		want    string
	}{
		{"user", "tiktok", Host + "/@tiktok"},
		{"video", "7212345678901234567", Host + "/@/video/7212345678901234567"},
		{"hashtag", "fyp", Host + "/tag/fyp"},
		{"sound", "6889283426862778117", Host + "/music/x-6889283426862778117"},
	}
	for _, c := range cases {
		got, err := Domain{}.Locate(c.uriType, c.id)
		if err != nil {
			t.Errorf("Locate(%q, %q) error: %v", c.uriType, c.id, err)
			continue
		}
		if got != c.want {
			t.Errorf("Locate(%q, %q) = %q, want %q", c.uriType, c.id, got, c.want)
		}
	}
	if _, err := (Domain{}).Locate("nope", "x"); err == nil {
		t.Fatal("Locate with unknown type want error, got nil")
	}
}

func TestClassifyLocateRoundTrip(t *testing.T) {
	// A handle URL classifies to (user, handle); Locate rebuilds the same URL,
	// and re-classifying it lands on the same resource.
	const link = "https://www.tiktok.com/@tiktok"
	typ, id, err := Domain{}.Classify(link)
	if err != nil {
		t.Fatal(err)
	}
	url, err := Domain{}.Locate(typ, id)
	if err != nil {
		t.Fatal(err)
	}
	typ2, id2, err := Domain{}.Classify(url)
	if err != nil {
		t.Fatal(err)
	}
	if typ2 != typ || id2 != id {
		t.Fatalf("round trip drifted: (%q,%q) -> %q -> (%q,%q)", typ, id, url, typ2, id2)
	}
}

func TestTagName(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"#fyp", "fyp", true},
		{"https://www.tiktok.com/tag/dance", "dance", true},
		{"fyp", "", false}, // a bare word is a handle, not a hashtag
		{"", "", false},
	}
	for _, c := range cases {
		got, ok := tagName(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("tagName(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestEffectiveLimit(t *testing.T) {
	if got := effectiveLimit(0, 35); got != 35 {
		t.Errorf("effectiveLimit(0, 35) = %d, want 35", got)
	}
	if got := effectiveLimit(10, 35); got != 10 {
		t.Errorf("effectiveLimit(10, 35) = %d, want 10", got)
	}
}
