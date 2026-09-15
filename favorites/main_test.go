package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderHomeFavorites(t *testing.T) {
	output := renderHome([]sdk.Page{{Slug: "guide/start", Title: "Guide & Start", Icon: "book-lucide", ViewCount: 9}}, fakeIcon)
	for _, expected := range []string{"Favorites", `href="/pages/guide/start"`, "Guide &amp; Start", "9 views", "[book-lucide]"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}

func TestRenderSidebarFavorites(t *testing.T) {
	output := renderSidebar([]sdk.Page{{Slug: "guide", Title: "Guide"}}, fakeIcon)
	if !strings.Contains(output, "Pinned") || !strings.Contains(output, "[star-lucide]") {
		t.Fatalf("unexpected sidebar: %s", output)
	}
}
