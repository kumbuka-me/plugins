package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransform(t *testing.T) {
	t.Parallel()

	t.Run("wraps Mermaid fences", func(t *testing.T) {
		t.Parallel()

		source := `<pre><code class="language-mermaid">flowchart LR
 A --&gt; B
</code></pre>`
		result := transform(sdk.RenderRequest{
			Module: "fences",
			Stage:  "postprocess",
			Source: source,
		})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, `data-kumbuka-plugin="me.kumbuka.mermaid"`)
		assert.Contains(t, result.Parts[0].Text, "flowchart LR\n A --&gt; B")
	})

	t.Run("preserves multiple diagrams and surrounding markup", func(t *testing.T) {
		t.Parallel()

		source := `<p>Before</p><pre><code class="language-mermaid">a</code></pre><p>Between</p><pre><code class="language-mermaid">b</code></pre><p>After</p>`
		result := transform(sdk.RenderRequest{
			Module: "fences",
			Stage:  "postprocess",
			Source: source,
		})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Contains(t, result.Parts[0].Text, "<p>Before</p>")
		assert.Contains(t, result.Parts[0].Text, "<p>Between</p>")
		assert.Contains(t, result.Parts[0].Text, "<p>After</p>")
		assert.Equal(t, 2, strings.Count(result.Parts[0].Text, `data-kumbuka-plugin="me.kumbuka.mermaid"`))
	})

	t.Run("consumes legacy static-site directive without exposing markup", func(t *testing.T) {
		t.Parallel()

		source := `<pre><code class="language-mermaid">flowchart LR</code></pre><p>{mermaid interactive=&quot;true&quot;}</p><p>After</p>`
		result := transform(sdk.RenderRequest{
			Module: "fences",
			Stage:  "postprocess",
			Source: source,
		})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.NotContains(t, result.Parts[0].Text, "{mermaid interactive=")
		assert.Contains(t, result.Parts[0].Text, "<p>After</p>")
	})

	t.Run("leaves other languages untouched", func(t *testing.T) {
		t.Parallel()

		source := `<pre><code class="language-go">fmt.Println(1)</code></pre>`
		result := transform(sdk.RenderRequest{
			Module: "fences",
			Stage:  "postprocess",
			Source: source,
		})

		require.Empty(t, result.Error)
		require.Len(t, result.Parts, 1)
		assert.Equal(t, source, result.Parts[0].Text)
	})

	t.Run("rejects an unsupported stage", func(t *testing.T) {
		t.Parallel()

		result := transform(sdk.RenderRequest{
			Module: "fences",
			Stage:  "preprocess",
			Source: "diagram",
		})

		assert.NotEmpty(t, result.Error)
	})
}
