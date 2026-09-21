package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestTransformDetails(t *testing.T) {
	result := transform(sdk.RenderRequest{Stage: "preprocess", Source: "Before\n\n???+ \"Show command\"\n\n    **Markdown**\n\nAfter"})
	require.Empty(t, result.Error)

	var text strings.Builder
	var markdown []string
	for _, part := range result.Parts {
		text.WriteString(part.Text)
		if part.Markdown != nil {
			markdown = append(markdown, *part.Markdown)
		}
	}

	require.Contains(t, text.String(), `<details class="markdown-details" open>`, "details markup missing: %s", text.String())
	require.Contains(t, text.String(), `<summary>Show command</summary>`, "details markup missing: %s", text.String())
	require.Len(t, markdown, 1, "unexpected details markdown: %#v", markdown)
	require.Contains(t, markdown[0], "**Markdown**", "unexpected details markdown: %#v", markdown)
}

func TestTransformDetailsIgnoresFences(t *testing.T) {
	source := "```text\n??? \"Not details\"\n```\n"
	result := transform(sdk.RenderRequest{Stage: "preprocess", Source: source})
	require.Len(t, result.Parts, 1, "fenced source changed: %#v", result.Parts)
	require.Equal(t, source, result.Parts[0].Text, "fenced source changed: %#v", result.Parts)
}
