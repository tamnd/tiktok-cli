// Package ttsign reimplements the request signing the TikTok web client adds to
// its /api/* calls: the msToken query parameter and the X-Bogus device
// signature, with the newer a_bogus variant behind the same Signer. It depends
// on nothing outside the standard library and is deterministic under an injected
// clock and randomness source, so the derivation can be unit tested.
//
// The signing is a faithful reimplementation of the public, logged-out web
// client behavior. It carries no account secret and reads only public data.
package ttsign

import (
	"net/url"
	"sort"
	"strings"
)

// Signer signs a query string. The clock and randomness are injectable so the
// output is reproducible.
type Signer struct {
	// NowMillis returns the current time in unix milliseconds.
	NowMillis func() int64
	// Rand returns n random bytes. It backs the msToken.
	Rand func(n int) []byte
	// MsTokenLen overrides the token length. Zero uses the default.
	MsTokenLen int
}

// Signed is the result of signing: the final query string with msToken appended
// and the headers to send.
type Signed struct {
	Query   string
	Headers map[string]string
}

// Sign appends a fresh msToken to rawQuery, computes X-Bogus over the result and
// the User-Agent, and returns the final query plus the headers to set. The
// caller adds Referer and Origin.
func (s *Signer) Sign(rawQuery, userAgent string) Signed {
	q := canonicalize(rawQuery)

	token := MsToken(s.MsTokenLen, s.Rand)
	if q != "" {
		q += "&"
	}
	q += "msToken=" + token

	xb := XBogus(q, userAgent, s.NowMillis()/1000)
	q += "&X-Bogus=" + xb

	return Signed{
		Query: q,
		Headers: map[string]string{
			"X-Bogus": xb,
		},
	}
}

// canonicalize sorts the query parameters by key, the order the client signs in.
// It keeps values url encoded.
func canonicalize(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		for j, v := range values[k] {
			if i > 0 || j > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(v))
		}
	}
	return b.String()
}
