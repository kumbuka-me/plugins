package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestTransformTabs(t *testing.T) {
	result := transform(sdk.RenderRequest{Stage: "preprocess", Source: "Before\n\n=== \"Linux\"\n\n    **apt**\n\n=== \"macOS\"\n\n    `brew`\n\nAfter"})
	require.Empty(t, result.Error)

	var text strings.Builder
	var markdown []string
	for _, part := range result.Parts {
		text.WriteString(part.Text)
		if part.Markdown != nil {
			markdown = append(markdown, *part.Markdown)
		}
	}

	require.Contains(t, text.String(), `class="markdown-tabs"`, "tab markup missing: %s", text.String())
	require.Contains(t, text.String(), `>Linux</button>`, "tab markup missing: %s", text.String())
	require.Len(t, markdown, 2, "unexpected panel markdown: %#v", markdown)
	require.Contains(t, markdown[0], "**apt**", "unexpected panel markdown: %#v", markdown)
	require.Contains(t, markdown[1], "`brew`", "unexpected panel markdown: %#v", markdown)
}

func TestTransformTabsIgnoresFences(t *testing.T) {
	source := "```text\n=== \"Not a tab\"\n```\n"
	result := transform(sdk.RenderRequest{Stage: "preprocess", Source: source})
	require.Len(t, result.Parts, 1, "fenced source changed: %#v", result.Parts)
	require.Equal(t, source, result.Parts[0].Text, "fenced source changed: %#v", result.Parts)
}
