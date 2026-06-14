package ttsign

import (
	"crypto/md5"
	"encoding/hex"
)

// ABogus reimplements the newer a_bogus signature, also surfaced as the
// X-Gnarly header on recent page builds. It signs the same query string plus an
// optional request body and the User-Agent, and is longer than X-Bogus because
// it folds in a small browser environment vector. The tool sends X-Bogus by
// default and carries this path for endpoints that stop accepting X-Bogus.
//
// The derivation reuses the shared primitives: the double MD5 of each input, a
// fixed environment vector, the timestamp, and the custom alphabet.
func ABogus(query, body, userAgent string, nowSeconds int64) string {
	q := doubleMD5Array(query)
	b := doubleMD5Array(body)
	u := doubleMD5Array(userAgent)

	// A fixed browser environment vector the client reports. The values are a
	// plausible desktop Chrome and do not need to vary per call.
	env := []int{1, 0, 1, 5, 1, 1, 1, 1}

	t := nowSeconds
	arr := make([]int, 0, 40)
	arr = append(arr, 64, 0, 1, 12)
	arr = append(arr, env...)
	arr = append(arr, q[14], q[15], b[14], b[15], u[14], u[15])
	arr = append(arr,
		int(t>>24)&255, int(t>>16)&255, int(t>>8)&255, int(t)&255,
		int(canvasConst>>24)&255, int(canvasConst>>16)&255,
		int(canvasConst>>8)&255, int(canvasConst)&255,
	)

	checksum := 0
	for _, v := range arr {
		checksum ^= v
	}
	arr = append(arr, checksum)

	payload := make([]byte, len(arr))
	for i, v := range arr {
		payload[i] = byte(v & 255)
	}
	payload = append([]byte{2, 255}, rc4([]byte{255}, payload)...)
	return customBase64(payload)
}

// md5Hex returns the hex md5 of s. It is exported-adjacent helper used by the
// a_bogus body hashing path and kept here so xbogus.go stays focused.
func md5Hex(s string) string {
	sum := md5.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

var _ = md5Hex // reserved for body digesting when an endpoint signs the POST body
