package main

import (
	"regexp"

	sdk "github.com/kumbuka-me/sdk"
)

// Goldmark's unhighlighted fence output has this shape. Capture its escaped
// source unchanged; the core sanitizer still processes the complete document.
var fence = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">.*?</code></pre>`)

// main runs the package entry point.
func main() {}

// init registers the Mermaid postprocessor with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("fences", transform) }

// transform marks Mermaid code blocks for the browser-side renderer.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}
	output := fence.ReplaceAllString(request.Source, `<div class="kumbuka-plugin-block" data-kumbuka-plugin="me.kumbuka.mermaid" data-kumbuka-module="diagrams">$0</div>`)
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}
