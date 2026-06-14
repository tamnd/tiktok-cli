package ttsign

import (
	"strings"
	"testing"
)

// fixedSigner returns a Signer with a frozen clock and deterministic bytes so
// the output is reproducible.
func fixedSigner() *Signer {
	var n byte
	return &Signer{
		NowMillis: func() int64 { return 1718323200000 },
		Rand: func(k int) []byte {
			b := make([]byte, k)
			for i := range b {
				n++
				b[i] = n
			}
			return b
		},
		MsTokenLen: 16,
	}
}

func TestSignDeterministic(t *testing.T) {
	s := fixedSigner()
	got := s.Sign("aid=1988&count=30&secUid=abc", DefaultUserAgentForTest)

	if !strings.Contains(got.Query, "msToken=") {
		t.Fatalf("query missing msToken: %q", got.Query)
	}
	if !strings.Contains(got.Query, "X-Bogus=") {
		t.Fatalf("query missing X-Bogus: %q", got.Query)
	}
	if got.Headers["X-Bogus"] == "" {
		t.Fatalf("missing X-Bogus header")
	}
	// The header value and the query value must match.
	if !strings.Contains(got.Query, "X-Bogus="+got.Headers["X-Bogus"]) {
		t.Fatalf("header and query X-Bogus disagree: %q vs %q", got.Headers["X-Bogus"], got.Query)
	}

	// Same inputs, same signer state reset: identical output.
	s2 := fixedSigner()
	got2 := s2.Sign("aid=1988&count=30&secUid=abc", DefaultUserAgentForTest)
	if got.Query != got2.Query {
		t.Fatalf("not deterministic:\n%q\n%q", got.Query, got2.Query)
	}
}

func TestCanonicalizeSorts(t *testing.T) {
	got := canonicalize("b=2&a=1&c=3")
	want := "a=1&b=2&c=3"
	if got != want {
		t.Fatalf("canonicalize = %q, want %q", got, want)
	}
}

func TestXBogusStable(t *testing.T) {
	a := XBogus("aid=1988", DefaultUserAgentForTest, 1718323200)
	b := XBogus("aid=1988", DefaultUserAgentForTest, 1718323200)
	if a != b {
		t.Fatalf("X-Bogus not stable: %q vs %q", a, b)
	}
	if a == "" {
		t.Fatal("X-Bogus empty")
	}
	// A different query yields a different signature.
	c := XBogus("aid=1989", DefaultUserAgentForTest, 1718323200)
	if a == c {
		t.Fatal("X-Bogus did not change with the query")
	}
}

func TestMsTokenShape(t *testing.T) {
	tok := MsToken(32, func(k int) []byte { return make([]byte, k) })
	if len(tok) != 32 {
		t.Fatalf("msToken length = %d, want 32", len(tok))
	}
	for _, r := range tok {
		if !strings.ContainsRune(msTokenAlphabet, r) {
			t.Fatalf("msToken has out-of-alphabet rune %q", r)
		}
	}
}

func TestABogusStable(t *testing.T) {
	a := ABogus("aid=1988", "", DefaultUserAgentForTest, 1718323200)
	b := ABogus("aid=1988", "", DefaultUserAgentForTest, 1718323200)
	if a != b || a == "" {
		t.Fatalf("a_bogus not stable: %q vs %q", a, b)
	}
}

// DefaultUserAgentForTest is a desktop Chrome UA used across the signing tests.
const DefaultUserAgentForTest = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"
