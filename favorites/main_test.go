package main

import (
	"testing"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderHomeFavorites(t *testing.T) {
	output := renderHome([]sdk.Page{{Slug: "guide/start", Title: "Guide & Start", Icon: "book-lucide", ViewCount: 9}}, fakeIcon, localize.For("en"))
	for _, expected := range []string{"Favorites", `href="/pages/guide/start"`, "Guide &amp; Start", "9 views", "[book-lucide]"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderSidebarFavorites(t *testing.T) {
	output := renderSidebar([]sdk.Page{{Slug: "guide", Title: "Guide"}}, fakeIcon, localize.For("en"))
	require.Contains(t, output, "Pinned", "unexpected sidebar: %s", output)
	require.Contains(t, output, "[star-lucide]", "unexpected sidebar: %s", output)
}

func TestRenderFavoritesGerman(t *testing.T) {
	output := renderHome([]sdk.Page{{Slug: "guide", Title: "Guide", ViewCount: 2}}, fakeIcon, localize.For("de"))
	require.Contains(t, output, "Favoriten")
	require.Contains(t, output, "2 Aufrufe")
	require.Contains(t, renderSidebar([]sdk.Page{{Slug: "guide", Title: "Guide"}}, fakeIcon, localize.For("de")), "Angeheftet")
}
