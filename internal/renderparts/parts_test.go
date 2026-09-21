package renderparts

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestAppendText(t *testing.T) {
	t.Parallel()

	t.Run("coalesces adjacent literal parts", func(t *testing.T) {
		t.Parallel()
		parts := []sdk.RenderPart{{Text: "a"}}
		AppendText(&parts, "b")
		if len(parts) != 1 || parts[0].Text != "ab" {
			t.Fatalf("AppendText() = %#v", parts)
		}
	})

	t.Run("preserves markdown boundary", func(t *testing.T) {
		t.Parallel()
		markdown := "body"
		parts := []sdk.RenderPart{{Markdown: &markdown}}
		AppendText(&parts, "tail")
		if len(parts) != 2 || parts[1].Text != "tail" {
			t.Fatalf("AppendText() = %#v", parts)
		}
	})

	t.Run("ignores empty text", func(t *testing.T) {
		t.Parallel()
		parts := []sdk.RenderPart{{Text: "a"}}
		AppendText(&parts, "")
		if len(parts) != 1 || parts[0].Text != "a" {
			t.Fatalf("AppendText() = %#v", parts)
		}
	})
}
