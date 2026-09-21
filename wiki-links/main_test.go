package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestRenderLinksEscapesAndMarksMissingTargets(t *testing.T) {
	output := renderLinks(sdk.PageLinks{
		Backlinks: []sdk.Page{{Slug: "guide/start", Title: `<Guide>`}},
		Outgoing: []sdk.PageLink{
			{TargetSlug: "api/auth", TargetTitle: `Auth & Login`, Exists: true},
			{TargetSlug: `missing page`, Exists: false},
		},
	})

	for _, expected := range []string{`href="/pages/guide/start"`, `&lt;Guide&gt;`, `Auth &amp; Login`, `href="/pages/new?slug=missing+page"`, `class="widget-row broken"`} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
	require.NotContains(t, output, `<Guide>`, "unescaped title in output: %s", output)
}

func TestRenderLinksEmptyStates(t *testing.T) {
	output := renderLinks(sdk.PageLinks{})
	require.Contains(t, output, "No pages link here yet.", "missing empty state: %s", output)
	require.Contains(t, output, "No wiki links on this page.", "missing empty state: %s", output)
}
