package ttsign

// msTokenAlphabet is the character set the web client draws an msToken from.
const msTokenAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

// DefaultMsTokenLen is the length of a freshly minted token. The real client
// mints a token of this order and the server rotates it through a Set-Cookie on
// the first answered call.
const DefaultMsTokenLen = 128

// MsToken returns an n character token drawn from the client's alphabet. The
// randomness source is injected so the output is reproducible under test. A
// logged-out call carries a freshly minted token as a bootstrap; the server
// accepts a well shaped one to start a session.
func MsToken(n int, rand func(int) []byte) string {
	if n <= 0 {
		n = DefaultMsTokenLen
	}
	raw := rand(n)
	out := make([]byte, n)
	for i := range n {
		out[i] = msTokenAlphabet[int(raw[i])%len(msTokenAlphabet)]
	}
	return string(out)
}
