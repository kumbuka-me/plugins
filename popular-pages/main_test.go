package main

import (
	"testing"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderPopular(t *testing.T) {
	output := renderPopular([]sdk.Page{{Slug: "guide", Title: "Guide", ViewCount: 42}}, fakeIcon, localize.For("en"))
	require.Contains(t, output, "Popular pages", "unexpected output: %s", output)
	require.Contains(t, output, "42 views", "unexpected output: %s", output)
}

func TestRenderPopularGerman(t *testing.T) {
	output := renderPopular([]sdk.Page{{Slug: "guide", Title: "Guide", ViewCount: 42}}, fakeIcon, localize.For("de"))
	require.Contains(t, output, "Beliebte Seiten")
	require.Contains(t, output, "42 Aufrufe")
}
