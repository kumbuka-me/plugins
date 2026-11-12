package main

import (
	"html"
	"regexp"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	pluginID = "me.kumbuka.mermaid"
	moduleID = "diagrams"
)

var (
	// mermaidFence locates escaped Mermaid code in intermediate Goldmark HTML.
	mermaidFence = regexp.MustCompile(`(?s)<pre><code class="language-mermaid">(.*?)</code></pre>`)
	// interactiveAfterFence recognizes the explicit Markdown directive after a code block.
	interactiveAfterFence = regexp.MustCompile(`(?is)^\s*<p>\s*\{mermaid\s+interactive=(?:"|&quot;|&#34;|&#x22;)(true|false)(?:"|&quot;|&#34;|&#x22;)\}\s*</p>`)
	// interactiveInSource recognizes the equivalent Mermaid comment directive.
	interactiveInSource = regexp.MustCompile(`(?mi)^%%[ \t]+kumbuka-interactive:[ \t]*true[ \t]*$`)
	widthInSource       = regexp.MustCompile(`(?mi)^%%[ \t]+kumbuka-width:[ \t]*([1-9][0-9]{0,3})[ \t]*\r?(?:\n|$)`)
	// alignInSource selects a purely presentational alignment for SVGs and browser diagrams.
	alignInSource = regexp.MustCompile(`(?mi)^%%[ \t]+kumbuka-align:[ \t]*(left|right|center)[ \t]*$`)
)

// main provides the plugin executable entry point.
func main() {}

// init registers the Mermaid postprocessor.
func init() { sdk.RegisterModule("fences", transform) }

// transform renders ordinary Mermaid fences to stored SVG pairs and preserves opt-in interactive diagrams.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "fences" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported Mermaid render request"}
	}
	rendered, err := transformWith(request.Source, sdk.Artifacts().Render)
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.Text(rendered)
}

// transformWith injects renderer and storage boundaries for independently testable transformations.
func transformWith(source string, publish func(sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error)) (string, error) {
	matches := mermaidFence.FindAllStringSubmatchIndex(source, -1)
	if len(matches) == 0 {
		return source, nil
	}

	var output strings.Builder
	position := 0
	for _, match := range matches {
		start, fenceEnd := match[0], match[1]
		diagram := html.UnescapeString(source[match[2]:match[3]])
		end := fenceEnd
		interactive := interactiveInSource.MatchString(diagram)
		if directive := interactiveAfterFence.FindStringSubmatchIndex(source[fenceEnd:]); directive != nil {
			interactive = strings.EqualFold(source[fenceEnd+directive[2]:fenceEnd+directive[3]], "true") || interactive
			end += directive[1]
		}

		output.WriteString(source[position:start])
		if interactive {
			output.WriteString(`<div class="kumbuka-plugin-block" data-kumbuka-plugin="` + pluginID + `" data-kumbuka-module="` + moduleID + `">`)
			output.WriteString(source[start:fenceEnd])
			output.WriteString(`</div>`)
		} else {
			markup, err := renderStatic(diagram, publish)
			if err != nil {
				return "", err
			}
			output.WriteString(markup)
		}
		position = end
	}
	output.WriteString(source[position:])
	return output.String(), nil
}

// renderStatic asks the generic host engine to execute this plugin's packaged SVG renderer.
func renderStatic(source string, publish func(sdk.RenderedArtifactRender) (sdk.RenderedArtifactRef, error)) (string, error) {
	widthAttribute := ""
	if match := widthInSource.FindStringSubmatch(source); match != nil {
		widthAttribute = ` style="width:` + match[1] + `px"`
		source = widthInSource.ReplaceAllString(source, "")
	}
	lightRef, err := publish(sdk.RenderedArtifactRender{Script: "headless.js", Source: source, Theme: "default"})
	if err != nil {
		return "", err
	}
	darkRef, err := publish(sdk.RenderedArtifactRender{Script: "headless.js", Source: source, Theme: "dark"})
	if err != nil {
		return "", err
	}

	alignment := "center"
	if match := alignInSource.FindStringSubmatch(source); len(match) > 1 {
		alignment = strings.ToLower(match[1])
	}
	return `<span class="kumbuka-rendered-artifact kumbuka-artifact-align-` + alignment + `">` +
		`<img class="kumbuka-artifact-light"` + widthAttribute + ` alt="Mermaid diagram" src="` + html.EscapeString(lightRef.URL) + `">` +
		`<img class="kumbuka-artifact-dark"` + widthAttribute + ` alt="Mermaid diagram" src="` + html.EscapeString(darkRef.URL) + `"></span>`, nil
}
