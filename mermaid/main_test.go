package main

import (
	"errors"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPublisher records the requested themes without needing a JavaScript runtime.
func testPublisher(request sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
	return sdk.RenderedArtifactRef{URL: "/assets/" + request.Theme + ".svg"}, nil
}

func TestTransform(t *testing.T) {
	t.Parallel()

	t.Run("renders ordinary fences to two passive images", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-mermaid">flowchart LR
 A --&gt; B</code></pre>`
		var themes []string
		output, err := transformWith(source, func(request sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
			require.Equal(t, "headless.js", request.Script)
			require.Contains(t, request.Source, "A --> B")
			themes = append(themes, request.Theme)
			return testPublisher(request)
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"default", "dark"}, themes)
		assert.Contains(t, output, `<img class="kumbuka-artifact-light"`)
		assert.Contains(t, output, `/assets/default.svg`)
		assert.Contains(t, output, `/assets/dark.svg`)
		assert.NotContains(t, output, `data-kumbuka-plugin=`)
	})

	t.Run("sizes both images while reusing the same artifact source", func(t *testing.T) {
		source := "<pre><code class=\"language-mermaid\">%% kumbuka-width: 320\r\nflowchart LR\nA --&gt; B</code></pre>"
		output, err := transformWith(source, func(request sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
			require.Equal(t, "flowchart LR\nA --> B", request.Source)
			return testPublisher(request)
		})
		require.NoError(t, err)
		assert.Equal(t, 2, strings.Count(output, `style="width:320px"`))
	})

	t.Run("interactive comment keeps the isolated browser module", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-mermaid">%% kumbuka-interactive: true
flowchart LR
A --&gt; B</code></pre>`
		output, err := transformWith(source, func(sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
			return sdk.RenderedArtifactRef{}, errors.New("must not render interactive diagrams")
		})
		require.NoError(t, err)
		assert.Contains(t, output, `data-kumbuka-plugin="me.kumbuka.mermaid"`)
	})

	t.Run("interactive directive is consumed", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-mermaid">graph LR; A--&gt;B</code></pre><p>{mermaid interactive=&quot;true&quot;}</p><p>After</p>`
		output, err := transformWith(source, func(sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
			return sdk.RenderedArtifactRef{}, errors.New("must not render interactive diagrams")
		})
		require.NoError(t, err)
		assert.Contains(t, output, `data-kumbuka-plugin=`)
		assert.NotContains(t, output, `{mermaid interactive=`)
		assert.Contains(t, output, `<p>After</p>`)
	})

	t.Run("explicit false still produces static images", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-mermaid">flowchart LR</code></pre><p>{mermaid interactive=&quot;false&quot;}</p>`
		output, err := transformWith(source, testPublisher)
		require.NoError(t, err)
		assert.Contains(t, output, `class="kumbuka-rendered-artifact`)
		assert.NotContains(t, output, `{mermaid interactive=`)
	})

	t.Run("reports rendering failures", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-mermaid">broken</code></pre>`
		_, err := transformWith(source, func(sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error) {
			return sdk.RenderedArtifactRef{}, errors.New("syntax error")
		})
		require.ErrorContains(t, err, "syntax error")
	})

	t.Run("preserves multiple diagrams and surrounding markup", func(t *testing.T) {
		t.Parallel()
		source := `<p>Before</p><pre><code class="language-mermaid">a</code></pre><p>Between</p><pre><code class="language-mermaid">b</code></pre><p>After</p>`
		output, err := transformWith(source, testPublisher)
		require.NoError(t, err)
		assert.Contains(t, output, "<p>Before</p>")
		assert.Contains(t, output, "<p>Between</p>")
		assert.Contains(t, output, "<p>After</p>")
		assert.Equal(t, 2, strings.Count(output, `class="kumbuka-rendered-artifact`))
	})

	t.Run("ignores other code languages", func(t *testing.T) {
		t.Parallel()
		source := `<pre><code class="language-go">fmt.Println(1)</code></pre>`
		output, err := transformWith(source, testPublisher)
		require.NoError(t, err)
		assert.Equal(t, source, output)
	})

	t.Run("rejects wrong module stage", func(t *testing.T) {
		t.Parallel()
		result := transform(sdk.RenderRequest{Module: "fences", Stage: "preprocess", Source: "diagram"})
		require.NotEmpty(t, result.Error)
	})
}
