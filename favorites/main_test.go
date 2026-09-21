package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderHomeFavorites(t *testing.T) {
	output := renderHome([]sdk.Page{{Slug: "guide/start", Title: "Guide & Start", Icon: "book-lucide", ViewCount: 9}}, fakeIcon)
	for _, expected := range []string{"Favorites", `href="/pages/guide/start"`, "Guide &amp; Start", "9 views", "[book-lucide]"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderSidebarFavorites(t *testing.T) {
	output := renderSidebar([]sdk.Page{{Slug: "guide", Title: "Guide"}}, fakeIcon)
	require.Contains(t, output, "Pinned", "unexpected sidebar: %s", output)
	require.Contains(t, output, "[star-lucide]", "unexpected sidebar: %s", output)
}
