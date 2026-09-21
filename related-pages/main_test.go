package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestRenderRelated(t *testing.T) {
	output := renderRelated([]sdk.Page{{Slug: "guide/start", Title: `Guide & Start`}})
	for _, expected := range []string{"Related pages", `href="/pages/guide/start"`, `Guide &amp; Start`} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderRelatedEmpty(t *testing.T) {
	output := renderRelated(nil)
	require.Contains(t, output, "No related pages yet.", "missing empty state: %s", output)
}
