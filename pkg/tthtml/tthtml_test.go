package tthtml

import "testing"

func TestScriptJSON(t *testing.T) {
	html := `<html><head>
<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__" type="application/json">{"a":1}</script>
</head></html>`
	got, err := ScriptJSON(html, UniversalDataID)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"a":1}` {
		t.Fatalf("got %q", got)
	}
}

func TestScriptJSONMissing(t *testing.T) {
	if _, err := ScriptJSON("<html></html>", UniversalDataID); err != ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestIsWAFChallenge(t *testing.T) {
	stub := `<html><body>Please wait... <p id="wci" class="_wafchallengeid"></p></body></html>`
	if !IsWAFChallenge(stub) {
		t.Fatal("expected WAF challenge to be detected")
	}
	real := `<html><script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{}</script>` + makeLong()
	if IsWAFChallenge(real) {
		t.Fatal("real page flagged as WAF challenge")
	}
}

func makeLong() string {
	b := make([]byte, 30000)
	for i := range b {
		b[i] = 'x'
	}
	return string(b)
}
