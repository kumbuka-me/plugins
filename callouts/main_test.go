package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

// TestCalloutFragments verifies callout fragments behavior.
func TestCalloutFragments(t *testing.T) {
	result := transform(sdk.RenderRequest{APIVersion: 1, Module: "callouts", Stage: "preprocess", Source: "Before\n\n!!! warning\n**Body**\n\nAfter\n"})
	require.Empty(t, result.Error)
	require.Len(t, result.Parts, 3, "unexpected fragments: %+v", result.Parts)
	require.NotNil(t, result.Parts[1].Markdown, "unexpected fragments: %+v", result.Parts)
	require.Equal(t, "**Body**", *result.Parts[1].Markdown, "unexpected fragments: %+v", result.Parts)
	require.Contains(t, result.Parts[0].Text, `<aside class="callout warning">`, "unexpected markup: %+v", result.Parts)
	require.Contains(t, result.Parts[2].Text, "After\n", "unexpected markup: %+v", result.Parts)
}

// TestCodeAndOrdinaryMarkdownRemainLiteral verifies code and ordinary markdown remain literal behavior.
func TestCodeAndOrdinaryMarkdownRemainLiteral(t *testing.T) {
	for _, source := range []string{"plain\n", "````\n```\n!!! warning\nBody\n`````\n", "~~~\n!!! note\nBody\n~~~", "!!! unsupported\nBody\n"} {
		result := transform(sdk.RenderRequest{APIVersion: 1, Module: "callouts", Stage: "preprocess", Source: source})
		require.Len(t, result.Parts, 1, "changed literal Markdown: %+v", result)
		require.Equal(t, source, result.Parts[0].Text, "changed literal Markdown: %+v", result)
	}
}

func TestCalloutBodyPreservesMarkdownIndentation(t *testing.T) {
	t.Parallel()

	result := transform(sdk.RenderRequest{
		APIVersion: 1,
		Module:     "callouts",
		Stage:      "preprocess",
		Source:     "!!! note\n- Parent\n  - Nested\n\n",
	})

	require.Empty(t, result.Error)
	require.Len(t, result.Parts, 3, "unexpected fragments: %+v", result.Parts)
	require.NotNil(t, result.Parts[1].Markdown, "unexpected fragments: %+v", result.Parts)
	require.Equal(t, "- Parent\n  - Nested", *result.Parts[1].Markdown)
}
