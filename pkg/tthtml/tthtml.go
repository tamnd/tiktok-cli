// Package tthtml pulls a named <script> JSON blob out of a TikTok web page. The
// page ships its state in <script id="..." type="application/json">...</script>
// tags; this package finds one by id and returns its raw JSON bytes.
package tthtml

import (
	"errors"
	"strings"
)

// ErrNotFound means the page did not contain a script with the requested id.
var ErrNotFound = errors.New("script blob not found")

// UniversalDataID is the id of the rehydration blob every modern TikTok page
// carries.
const UniversalDataID = "__UNIVERSAL_DATA_FOR_REHYDRATION__"

// ScriptJSON returns the raw JSON text inside <script id="{id}" ...>...</script>.
// It scans the markup directly, which is cheaper and more forgiving than a full
// HTML parse for a single well known tag.
func ScriptJSON(html, id string) (string, error) {
	marker := `id="` + id + `"`
	idx := strings.Index(html, marker)
	if idx < 0 {
		return "", ErrNotFound
	}
	// Find the end of the opening tag.
	open := strings.IndexByte(html[idx:], '>')
	if open < 0 {
		return "", ErrNotFound
	}
	start := idx + open + 1
	end := strings.Index(html[start:], "</script>")
	if end < 0 {
		return "", ErrNotFound
	}
	return strings.TrimSpace(html[start : start+end]), nil
}

// IsWAFChallenge reports whether a page body is the SlardarWAF "Please wait"
// challenge stub rather than real content. The stub is short and carries the
// challenge markers.
func IsWAFChallenge(html string) bool {
	if len(html) > 20000 {
		return false
	}
	return strings.Contains(html, "_wafchallengeid") ||
		strings.Contains(html, "Please wait...") ||
		strings.Contains(html, "SlardarWAF")
}
