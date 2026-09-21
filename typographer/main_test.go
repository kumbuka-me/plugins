package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestTypographerOwnsPunctuationTransformation(t *testing.T) {
	got, err := typographHTML(`<p>"Kumbuka" -- documentation... It's useful. '90s.</p>`, false)
	require.NoError(t, err)
	for _, want := range []string{"“Kumbuka”", "– documentation…", "It’s useful.", "’90s"} {
		require.Contains(t, got, want, "missing %q in %s", want, got)
	}
}

func TestTypographerTracksQuotesAcrossInlineMarkupAndSkipsCode(t *testing.T) {
	got, err := typographHTML(`<p>"<em>Kumbuka</em>" '<strong>plugin</strong>' <code>"-- ..."</code></p>`, false)
	require.NoError(t, err)
	for _, want := range []string{`“<em>Kumbuka</em>”`, `‘<strong>plugin</strong>’`} {
		require.Contains(t, got, want, "inline quote pair %q was not preserved: %s", want, got)
	}
	require.Contains(t, got, `<code>"-- ..."</code>`, "code was transformed: %s", got)
}

func TestTypographerHonorsProgrammingOperatorPolicy(t *testing.T) {
	features := map[string]bool{sdk.RenderPolicyFeature(preserveOperatorsPolicy): true}
	result := transform(sdk.RenderRequest{
		Module:   "typographer",
		Stage:    "postprocess",
		Source:   `<p>"quoted" --> -- --- << >> ...</p>`,
		Features: features,
	})
	require.Empty(t, result.Error)
	got := result.Parts[0].Text
	for _, want := range []string{"“quoted”", "--&gt; -- --- &lt;&lt; &gt;&gt; …"} {
		require.Contains(t, got, want, "missing %q in %s", want, got)
	}
}
