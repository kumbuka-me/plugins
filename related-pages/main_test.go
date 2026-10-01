package main

import (
	"testing"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestRenderRelated(t *testing.T) {
	output := renderRelated([]sdk.Page{{URL: "/p/7/guide/start", Slug: "guide/start", Title: `Guide & Start`}}, localize.For("en"))
	for _, expected := range []string{"Related pages", `href="/p/7/guide/start"`, `Guide &amp; Start`} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderRelatedEmpty(t *testing.T) {
	output := renderRelated(nil, localize.For("en"))
	require.Contains(t, output, "No related pages yet.", "missing empty state: %s", output)
}

func TestRenderRelatedGerman(t *testing.T) {
	output := renderRelated(nil, localize.For("de"))
	require.Contains(t, output, "Verwandte Seiten")
	require.Contains(t, output, "Noch keine verwandten Seiten.")
}
