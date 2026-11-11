package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMermaidTransform(t *testing.T) {
	t.Parallel()

	t.Run("wraps ordinary fences without changing source", func(t *testing.T) {
		t.Parallel()

		source := "<pre><code class=\"language-mermaid\">flowchart LR\n A --&gt; B\n</code></pre>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, `data-kumbuka-plugin="me.kumbuka.mermaid"`)
		assert.Contains(t, result.Parts[0].Text, "flowchart LR\n A --&gt; B")
		assert.NotContains(t, result.Parts[0].Text, "kumbuka-interactive")
	})

	t.Run("consumes true directive and keeps interaction marker", func(t *testing.T) {
		t.Parallel()

		source := "<pre><code class=\"language-mermaid\">flowchart LR\nA --&gt; B</code></pre>\n<p>{mermaid interactive=&#34;true&#34;}</p>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, "%% kumbuka-interactive: true\nflowchart LR")
		assert.NotContains(t, result.Parts[0].Text, "{mermaid interactive=")
	})

	t.Run("handles typographer quotes in interactive directive", func(t *testing.T) {
		t.Parallel()

		source := "<pre><code class=\"language-mermaid\">flowchart TD</code></pre><p>{mermaid interactive=“true”}</p>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, "%% kumbuka-interactive: true\n")
		assert.NotContains(t, result.Parts[0].Text, "{mermaid interactive=")
	})

	t.Run("consumes false directive without interaction marker", func(t *testing.T) {
		t.Parallel()

		source := "<pre><code class=\"language-mermaid\">flowchart LR</code></pre><p>{mermaid interactive=&quot;false&quot;}</p>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.NotContains(t, result.Parts[0].Text, "kumbuka-interactive")
		assert.NotContains(t, result.Parts[0].Text, "{mermaid")
	})

	t.Run("keeps ordinary content and consecutive diagrams", func(t *testing.T) {
		t.Parallel()

		source := "<p>Before</p><pre><code class=\"language-mermaid\">a</code></pre><p>Between</p><pre><code class=\"language-mermaid\">b</code></pre><p>After</p>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, "<p>Before</p>")
		assert.Contains(t, result.Parts[0].Text, "<p>Between</p>")
		assert.Contains(t, result.Parts[0].Text, "<p>After</p>")
		assert.Equal(t, 2, countBlocks(result.Parts[0].Text))
	})

	t.Run("keeps non-Mermaid content untouched", func(t *testing.T) {
		t.Parallel()

		source := "<pre><code class=\"language-go\">fmt.Println(1)</code></pre>"
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "postprocess", Source: source})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Equal(t, source, result.Parts[0].Text)
	})

	t.Run("rejects other stages", func(t *testing.T) {
		t.Parallel()

		result := transform(sdk.RenderRequest{Module: "fences", Stage: "preprocess", Source: "diagram"})

		assert.NotEmpty(t, result.Error)
	})
}

func countBlocks(source string) int {
	return len(mermaidFence.FindAllStringIndex(source, -1))
}
