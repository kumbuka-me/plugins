package main

import (
	"regexp"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	pluginID           = "me.kumbuka.mermaid"
	moduleID           = "diagrams"
	interactiveComment = "%% kumbuka-interactive: true\n"
)

var (
	// Goldmark preserves the escaped source inside an unhighlighted Mermaid fence.
	mermaidFence = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)
	// The standalone directive is optional; an in-fence comment is editor-friendly.
	mermaidDirective = regexp.MustCompile(`(?i)^\s*<p>\s*\{mermaid\s+interactive=(?:"|&quot;|&#34;|&#x22;|“|”|&ldquo;|&rdquo;|&#8220;|&#8221;)(true|false)(?:"|&quot;|&#34;|&#x22;|“|”|&ldquo;|&rdquo;|&#8220;|&#8221;)\}\s*</p>`)
)

// main runs the package entry point.
func main() {}

// init registers the Mermaid postprocessor with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("fences", transform) }

// transform wraps Mermaid code blocks and consumes optional interactive directives.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported Mermaid render request"}
	}

	matches := mermaidFence.FindAllStringSubmatchIndex(request.Source, -1)
	if len(matches) == 0 {
		return sdk.Text(request.Source)
	}

	var output strings.Builder
	position := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		code := request.Source[match[2]:match[3]]
		rest := request.Source[end:]
		if directive := mermaidDirective.FindStringSubmatchIndex(rest); directive != nil {
			if strings.EqualFold(rest[directive[2]:directive[3]], "true") {
				code = interactiveComment + code
			}
			end += directive[1]
		}

		output.WriteString(request.Source[position:start])
		output.WriteString(`<div class="kumbuka-plugin-block" data-kumbuka-plugin="` + pluginID + `" data-kumbuka-module="` + moduleID + `"><pre><code class="language-mermaid">`)
		output.WriteString(code)
		output.WriteString(`</code></pre></div>`)
		position = end
	}
	output.WriteString(request.Source[position:])

	return sdk.Text(output.String())
}
