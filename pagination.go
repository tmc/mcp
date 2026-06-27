package mcp

import (
	"encoding/base64"
	"sort"
)

// defaultPageSize bounds how many items a single list response returns when the
// client does not constrain it. Large registries would otherwise return their
// entire contents in one message.
const defaultPageSize = 100

// paginate returns a stable page of items together with an opaque cursor for the
// next page. Items are ordered by key(item); the page begins just after the
// item identified by the incoming cursor and contains at most defaultPageSize
// items. next is empty when the page reaches the end of the list.
//
// The cursor is opaque to clients (as the MCP spec requires) but encodes the key
// of the last item returned, so paging is stable across calls even though the
// underlying registry is a Go map with unspecified iteration order. An unknown
// or malformed cursor yields an empty page with no error, matching the spec's
// "the server MAY return an error" latitude with the least surprising behavior.
func paginate[T any](items []T, cursor string, key func(T) string) (page []T, next string) {
	sort.Slice(items, func(i, j int) bool { return key(items[i]) < key(items[j]) })

	start := 0
	if cursor != "" {
		after, ok := decodeCursor(cursor)
		if !ok {
			return nil, ""
		}
		// First item whose key is strictly greater than the cursor key.
		start = sort.Search(len(items), func(i int) bool { return key(items[i]) > after })
	}

	end := start + defaultPageSize
	if end >= len(items) {
		return items[start:], ""
	}
	return items[start:end], encodeCursor(key(items[end-1]))
}

func encodeCursor(key string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(key))
}

func decodeCursor(cursor string) (string, bool) {
	b, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return "", false
	}
	return string(b), true
}
