package main

import (
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderHomeRecentlyViewed(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderHome([]sdk.Page{{Slug: "guide", Title: "Guide", UpdatedAt: now.Add(-2 * time.Hour)}}, now, fakeIcon)
	for _, expected := range []string{"Recently viewed", `href="/pages/guide"`, "2h ago"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestWithoutFavorites(t *testing.T) {
	pages := []sdk.Page{{Slug: "one"}, {Slug: "two"}, {Slug: "three"}}
	favorites := []sdk.Page{{Slug: "two"}}
	result := withoutFavorites(pages, favorites, 2)
	require.Len(t, result, 2, "unexpected pages: %+v", result)
	require.Equal(t, "one", result[0].Slug, "unexpected pages: %+v", result)
	require.Equal(t, "three", result[1].Slug, "unexpected pages: %+v", result)
}
