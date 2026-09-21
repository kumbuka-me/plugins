package renderparts

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestAppendText(t *testing.T) {
	t.Parallel()

	t.Run("coalesces adjacent literal parts", func(t *testing.T) {
		t.Parallel()
		parts := []sdk.RenderPart{{Text: "a"}}
		AppendText(&parts, "b")
		require.Len(t, parts, 1, "AppendText() = %#v", parts)
		require.Equal(t, "ab", parts[0].Text, "AppendText() = %#v", parts)
	})

	t.Run("preserves markdown boundary", func(t *testing.T) {
		t.Parallel()
		markdown := "body"
		parts := []sdk.RenderPart{{Markdown: &markdown}}
		AppendText(&parts, "tail")
		require.Len(t, parts, 2, "AppendText() = %#v", parts)
		require.Equal(t, "tail", parts[1].Text, "AppendText() = %#v", parts)
	})

	t.Run("ignores empty text", func(t *testing.T) {
		t.Parallel()
		parts := []sdk.RenderPart{{Text: "a"}}
		AppendText(&parts, "")
		require.Len(t, parts, 1, "AppendText() = %#v", parts)
		require.Equal(t, "a", parts[0].Text, "AppendText() = %#v", parts)
	})
}
