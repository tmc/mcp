package mcp

import (
	"fmt"
	"testing"
)

func identity(s string) string { return s }

func TestPaginate(t *testing.T) {
	// Build an out-of-order input to prove paginate sorts before slicing.
	items := []string{"c", "a", "e", "b", "d"}

	page, next := paginate(items, "", identity)
	if got := fmt.Sprint(page); got != "[a b c d e]" {
		t.Fatalf("first page = %s, want sorted full set", got)
	}
	if next != "" {
		t.Fatalf("next cursor = %q, want empty (fits one page)", next)
	}
}

func TestPaginateWalk(t *testing.T) {
	// More items than one page forces multiple cursors.
	const total = defaultPageSize*2 + 7
	items := make([]string, total)
	for i := range items {
		// Zero-padded so lexical order matches numeric order.
		items[i] = fmt.Sprintf("k%05d", total-1-i) // reversed input
	}

	seen := make(map[string]int)
	cursor := ""
	pages := 0
	for {
		page, next := paginate(items, cursor, identity)
		if len(page) == 0 {
			t.Fatalf("empty page at cursor %q before exhausting items", cursor)
		}
		if len(page) > defaultPageSize {
			t.Fatalf("page of %d exceeds defaultPageSize %d", len(page), defaultPageSize)
		}
		// Pages must be internally sorted and continue from the previous page.
		for i := 1; i < len(page); i++ {
			if page[i-1] >= page[i] {
				t.Fatalf("page not sorted: %q >= %q", page[i-1], page[i])
			}
		}
		for _, k := range page {
			seen[k]++
		}
		pages++
		if next == "" {
			break
		}
		cursor = next
		if pages > total {
			t.Fatal("pagination did not terminate")
		}
	}

	if pages != 3 {
		t.Fatalf("walked %d pages, want 3 for %d items at page size %d", pages, total, defaultPageSize)
	}
	if len(seen) != total {
		t.Fatalf("saw %d distinct items, want %d", len(seen), total)
	}
	for k, n := range seen {
		if n != 1 {
			t.Fatalf("item %q returned %d times, want exactly 1", k, n)
		}
	}
}

func TestPaginateExactBoundary(t *testing.T) {
	// Exactly one full page: there is no more data, so next must be empty
	// rather than pointing at an empty trailing page.
	items := make([]string, defaultPageSize)
	for i := range items {
		items[i] = fmt.Sprintf("k%05d", i)
	}
	page, next := paginate(items, "", identity)
	if len(page) != defaultPageSize {
		t.Fatalf("page length = %d, want %d", len(page), defaultPageSize)
	}
	if next != "" {
		t.Fatalf("next cursor = %q, want empty at exact boundary", next)
	}
}

func TestPaginateUnknownCursor(t *testing.T) {
	items := []string{"a", "b", "c"}
	// A malformed (non-base64) cursor yields an empty page, no panic.
	page, next := paginate(items, "!!!not-base64!!!", identity)
	if len(page) != 0 || next != "" {
		t.Fatalf("malformed cursor = (%v, %q), want (empty, empty)", page, next)
	}
}

func TestPaginateCursorPastEnd(t *testing.T) {
	items := []string{"a", "b", "c"}
	// A valid cursor whose key sorts after every item returns an empty page.
	page, next := paginate(items, encodeCursor("z"), identity)
	if len(page) != 0 || next != "" {
		t.Fatalf("cursor past end = (%v, %q), want (empty, empty)", page, next)
	}
}
