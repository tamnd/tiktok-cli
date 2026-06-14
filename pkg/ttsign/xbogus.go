package ttsign

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
)

// character is the custom base64 alphabet the web client maps the final byte
// array through. It is the published TikTok web alphabet, not standard base64.
const character = "Dkdpgh4ZKsQB80/Mfvw36XI1R25-WUAlEi7NLboqYTOPuzmFjJnryx9HVGcaStCe="

// uaRC4Key is the fixed three byte key the client RC4s the User-Agent with
// before folding it into the signature.
var uaRC4Key = []byte{0x00, 0x01, 0x0e}

// canvasConst is the fixed canvas fingerprint constant the client mixes in.
const canvasConst = 536919696

// XBogus reimplements the web client's X-Bogus derivation. The value is a
// function of the final sorted query string, the User-Agent, and a one second
// timestamp. The steps mirror the public algorithm: hash the query with a
// double MD5, hash the RC4 then base64 of the User-Agent the same way, fold both
// with the timestamp and the canvas constant into a byte array, checksum it,
// interleave it, RC4 it under a single byte key, and map the result through the
// custom alphabet.
//
// nowSeconds is injected so the output is reproducible under test.
func XBogus(query, userAgent string, nowSeconds int64) string {
	paramsArr := doubleMD5Array(query)

	uaEncrypted := rc4(uaRC4Key, []byte(userAgent))
	uaB64 := base64.StdEncoding.EncodeToString(uaEncrypted)
	uaArr := doubleMD5Array(uaB64)

	t := nowSeconds
	ct := int64(canvasConst)

	// The folded array: a fixed header, two bytes each from the query and UA
	// digests, the four timestamp bytes, and the four canvas bytes.
	arr := []int{
		64, 0, 1, 12,
		paramsArr[14], paramsArr[15],
		uaArr[14], uaArr[15],
		int(t>>24) & 255, int(t>>16) & 255, int(t>>8) & 255, int(t) & 255,
		int(ct>>24) & 255, int(ct>>16) & 255, int(ct>>8) & 255, int(ct) & 255,
	}

	// XOR checksum across the whole array becomes the final byte.
	checksum := 0
	for _, v := range arr {
		checksum ^= v
	}
	arr = append(arr, checksum)

	// Interleave the header half with the timestamp/canvas half, the order the
	// client packs the bytes in before encrypting.
	order := []int{
		arr[0], arr[8], arr[1], arr[9],
		arr[2], arr[10], arr[3], arr[11],
		arr[4], arr[12], arr[5], arr[13],
		arr[6], arr[14], arr[7], arr[15],
		arr[16],
	}

	garbled := make([]byte, len(order))
	for i, v := range order {
		garbled[i] = byte(v & 255)
	}

	// A two byte prefix then a single byte RC4 key, the client's last shuffle
	// before the custom base64.
	payload := append([]byte{2, 255}, rc4([]byte{255}, garbled)...)

	return customBase64(payload)
}

// doubleMD5Array returns the 16 byte array of md5(md5(s)). The web client treats
// the first MD5 as a hex string fed back through md5, which is what hashHexThen
// bytes reproduces.
func doubleMD5Array(s string) []int {
	first := md5.Sum([]byte(s))
	firstHex := hex.EncodeToString(first[:])
	second := md5.Sum([]byte(firstHex))
	out := make([]int, len(second))
	for i, b := range second {
		out[i] = int(b)
	}
	return out
}

// customBase64 maps a byte slice through the client's alphabet, three input
// bytes to four output characters, the same packing as standard base64 with a
// substituted table.
func customBase64(data []byte) string {
	var out []byte
	for i := 0; i < len(data); i += 3 {
		var n int
		var pad int
		switch {
		case i+2 < len(data):
			n = int(data[i])<<16 | int(data[i+1])<<8 | int(data[i+2])
		case i+1 < len(data):
			n = int(data[i])<<16 | int(data[i+1])<<8
			pad = 1
		default:
			n = int(data[i]) << 16
			pad = 2
		}
		quad := []int{(n >> 18) & 63, (n >> 12) & 63, (n >> 6) & 63, n & 63}
		for j := 0; j < 4-pad; j++ {
			out = append(out, character[quad[j]])
		}
	}
	return string(out)
}

// rc4 is the standard stream cipher the client uses in two places.
func rc4(key, data []byte) []byte {
	s := make([]byte, 256)
	for i := range s {
		s[i] = byte(i)
	}
	j := 0
	for i := range 256 {
		j = (j + int(s[i]) + int(key[i%len(key)])) % 256
		s[i], s[j] = s[j], s[i]
	}
	out := make([]byte, len(data))
	i, j := 0, 0
	for k := range data {
		i = (i + 1) % 256
		j = (j + int(s[i])) % 256
		s[i], s[j] = s[j], s[i]
		out[k] = data[k] ^ s[(int(s[i])+int(s[j]))%256]
	}
	return out
}
