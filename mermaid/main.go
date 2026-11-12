package main

import (
	"regexp"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	pluginID = "me.kumbuka.mermaid"
	moduleID = "diagrams"
)

var (
	// mermaidFence identifies escaped Mermaid source produced by Goldmark.
	mermaidFence = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)
	// legacyInteractiveDirective consumes an obsolete static-renderer option without showing it as text.
	legacyInteractiveDirective = regexp.MustCompile(`(?i)^\s*<p>\s*\{mermaid\s+interactive=(?:"|&quot;|&#34;|&#x22;|“|”|&ldquo;|&rdquo;|&#8220;|&#8221;)(?:true|false)(?:"|&quot;|&#34;|&#x22;|“|”|&ldquo;|&rdquo;|&#8220;|&#8221;)\}\s*</p>`)
)

// main runs the package entry point.
func main() {}

// init registers the Mermaid postprocessor with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("fences", transform) }

// transform marks Mermaid fences for the browser renderer and consumes obsolete directives.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}

	matches := mermaidFence.FindAllStringIndex(request.Source, -1)
	if len(matches) == 0 {
		return sdk.Text(request.Source)
	}

	var output strings.Builder
	position := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if directive := legacyInteractiveDirective.FindStringIndex(request.Source[end:]); directive != nil {
			end += directive[1]
		}

		output.WriteString(request.Source[position:start])
		output.WriteString(`<div class="kumbuka-plugin-block" data-kumbuka-plugin="` + pluginID + `" data-kumbuka-module="` + moduleID + `">`)
		output.WriteString(request.Source[start:match[1]])
		output.WriteString(`</div>`)
		position = end
	}
	output.WriteString(request.Source[position:])

	return sdk.Text(output.String())
}
