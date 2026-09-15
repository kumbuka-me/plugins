package main

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderHomeRecentlyViewed(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderHome([]sdk.Page{{Slug: "guide", Title: "Guide", UpdatedAt: now.Add(-2 * time.Hour)}}, now, fakeIcon)
	for _, expected := range []string{"Recently viewed", `href="/pages/guide"`, "2h ago"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}

func TestWithoutFavorites(t *testing.T) {
	pages := []sdk.Page{{Slug: "one"}, {Slug: "two"}, {Slug: "three"}}
	favorites := []sdk.Page{{Slug: "two"}}
	result := withoutFavorites(pages, favorites, 2)
	if len(result) != 2 || result[0].Slug != "one" || result[1].Slug != "three" {
		t.Fatalf("unexpected pages: %+v", result)
	}
}
