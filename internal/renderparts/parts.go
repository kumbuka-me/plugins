// Package renderparts provides shared result-building helpers for first-party plugins.
package renderparts

import sdk "github.com/kumbuka-me/sdk"

// AppendText appends literal output while coalescing adjacent text fragments.
func AppendText(parts *[]sdk.RenderPart, text string) {
	if text == "" {
		return
	}
	if len(*parts) != 0 && (*parts)[len(*parts)-1].Markdown == nil {
		(*parts)[len(*parts)-1].Text += text
		return
	}
	*parts = append(*parts, sdk.RenderPart{Text: text})
}
