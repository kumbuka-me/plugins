// Callouts is a standalone WASI plugin. It uses the public Go SDK and wire types,
// never Kumbuka's renderer, registry, persistence, or handler packages.
package main

import (
	"html"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

// main runs the package entry point.
func main() {}

// init registers the callout renderer with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("callouts", transform) }

// transform converts supported callout blocks into intermediate HTML fragments.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "callouts" || request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported render request"}
	}
	lines := strings.Split(request.Source, "\n")
	var output fragments
	for index := 0; index < len(lines); index++ {
		if marker := pluginmarkdown.Fence(lines[index]); marker != "" {
			output.line(lines[index])
			for index++; index < len(lines); index++ {
				output.line(lines[index])
				if pluginmarkdown.Closes(lines[index], marker) {
					break
				}
			}
			continue
		}
		kind := calloutKind(lines[index])
		if kind == "" {
			output.line(lines[index])
			continue
		}
		var body []string
		for index++; index < len(lines) && strings.TrimSpace(lines[index]) != ""; index++ {
			body = append(body, strings.TrimSpace(lines[index]))
		}
		label := strings.ToUpper(kind[:1]) + kind[1:]
		output.line(`<aside class="callout ` + kind + `"><strong>` + html.EscapeString(label) + `</strong><div class="callout-body">`)
		markdown := strings.Join(body, "\n")
		output.flush()
		output.parts = append(output.parts, sdk.RenderPart{Markdown: &markdown})
		output.text("</div></aside>\n")
	}
	output.flush()
	return sdk.RenderResult{Parts: output.parts}
}

// calloutKind normalizes and validates a supported callout kind.
func calloutKind(line string) string {
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "!!! ")
	if !ok {
		return ""
	}
	fields := strings.Fields(body)
	if len(fields) == 0 {
		return ""
	}
	kind := strings.ToLower(fields[0])
	switch kind {
	case "note", "info", "tip", "success", "warning", "danger", "error":
		return kind
	default:
		return ""
	}
}

// fragments incrementally builds bounded render fragments for one callout transformation.
type fragments struct {
	// pending buffers literal text until the next structured fragment.
	pending strings.Builder
	// parts contains completed render fragments in output order.
	parts []sdk.RenderPart
	// lines tracks buffered literal line count for output limits.
	lines int
}

// line appends one literal output line.
func (f *fragments) line(value string) {
	if f.lines > 0 {
		f.text("\n")
	}
	f.text(value)
	f.lines++
}

// text appends one literal text fragment.
func (f *fragments) text(value string) { f.pending.WriteString(value) }

// flush emits buffered literal text as one fragment.
func (f *fragments) flush() {
	if f.pending.Len() == 0 {
		return
	}
	f.parts = append(f.parts, sdk.RenderPart{Text: f.pending.String()})
	f.pending.Reset()
}

// openingFence recognizes an opening Markdown fence and returns its marker.
// validFenceInfo validates the optional info string on a fenced code block.
// closesFence reports whether a line closes the active Markdown fence.
