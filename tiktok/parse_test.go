package tiktok

import "testing"

// videoPage is a trimmed but real-shaped rehydration blob for one video. It
// carries the fields the parser reads so the mapping is exercised without a
// 400 KB fixture.
const videoPage = `<!doctype html><html><body>
<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">
{"__DEFAULT_SCOPE__":{"webapp.video-detail":{"itemInfo":{"itemStruct":{
  "id":"7106594312292453675",
  "desc":"how many frogs did you find?",
  "createTime":1654000000,
  "author":{"id":"107955","uniqueId":"tiktok","nickname":"TikTok","secUid":"MS4wLjABAAAAv7iSuu"},
  "music":{"id":"6745","title":"original sound","authorName":"TikTok","duration":24},
  "challenges":[{"title":"Minecraft"}],
  "video":{"duration":24,"cover":"https://cover","playAddr":"https://play","downloadAddr":"https://dl","width":576,"height":1024},
  "stats":{"diggCount":98700,"shareCount":127,"commentCount":1292,"playCount":562500},
  "statsV2":{"collectCount":"58630"}
}}}}}
</script></body></html>`

func TestParseVideoBlob(t *testing.T) {
	u, err := parseUniversal(videoPage)
	if err != nil {
		t.Fatal(err)
	}
	v := videoFrom(u.DefaultScope.VideoDetail.ItemInfo.ItemStruct)

	if v.ID != "7106594312292453675" {
		t.Errorf("id = %q", v.ID)
	}
	if v.Author != "tiktok" || v.AuthorSecUID != "MS4wLjABAAAAv7iSuu" {
		t.Errorf("author = %q secUid = %q", v.Author, v.AuthorSecUID)
	}
	if v.DiggCount != 98700 || v.CommentCount != 1292 || v.PlayCount != 562500 {
		t.Errorf("stats wrong: %+v", v)
	}
	// collectCount comes from statsV2 as a string and must fall back correctly.
	if v.CollectCount != 58630 {
		t.Errorf("collectCount = %d, want 58630 (statsV2 string fallback)", v.CollectCount)
	}
	if len(v.Challenges) != 1 || v.Challenges[0] != "Minecraft" {
		t.Errorf("challenges = %v", v.Challenges)
	}
	if v.URL != "https://www.tiktok.com/@tiktok/video/7106594312292453675" {
		t.Errorf("url = %q", v.URL)
	}
}

func TestFlexInt(t *testing.T) {
	var f flexInt
	if err := f.UnmarshalJSON([]byte(`"123"`)); err != nil || f != 123 {
		t.Errorf("string number: %v %d", err, f)
	}
	if err := f.UnmarshalJSON([]byte(`456`)); err != nil || f != 456 {
		t.Errorf("bare number: %v %d", err, f)
	}
	if err := f.UnmarshalJSON([]byte(`null`)); err != nil || f != 0 {
		t.Errorf("null: %v %d", err, f)
	}
	if err := f.UnmarshalJSON([]byte(`""`)); err != nil || f != 0 {
		t.Errorf("empty string: %v %d", err, f)
	}
}
