// Callouts is a standalone WASI plugin. It uses the public Go SDK and wire types,
// never Kumbuka's renderer, registry, persistence, or handler packages.
package main

import (
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
	if !strings.Contains(request.Source, "!!! ") {
		return sdk.Text(request.Source)
	}

	output := fragments{parts: make([]sdk.RenderPart, 0, strings.Count(request.Source, "!!! ")*2+1)}
	for position := 0; position <= len(request.Source); {
		line, next, done := sourceLine(request.Source, position)

		if possibleFence(line) {
			if marker := pluginmarkdown.Fence(line); marker != "" {
				output.line(line)
				position = next
				for !done && position <= len(request.Source) {
					line, next, done = sourceLine(request.Source, position)
					output.line(line)
					position = next
					if pluginmarkdown.Closes(line, marker) {
						break
					}
				}
				if done {
					break
				}
				continue
			}
		}

		kind := calloutKind(line)
		if kind == "" {
			output.line(line)
			position = next
			if done {
				break
			}
			continue
		}

		markdown, nextPosition, atEnd := calloutBody(request.Source, next, done)
		position, done = nextPosition, atEnd

		output.line(`<aside class="callout ` + kind + `"><div class="callout-body">`)
		output.flush()
		output.parts = append(output.parts, sdk.RenderPart{Markdown: &markdown})
		output.text("</div></aside>\n")

		if done {
			break
		}
	}
	output.flush()
	return sdk.RenderResult{Parts: output.parts}
}

// calloutBody collects the indented Markdown body without losing empty lines.
func calloutBody(source string, position int, done bool) (string, int, bool) {
	var body strings.Builder
	firstLine := true

	for !done && position <= len(source) {
		line, next, lineDone := sourceLine(source, position)
		content, indented := deindent(line)
		if !indented && strings.TrimSpace(line) != "" {
			break
		}
		if !firstLine {
			body.WriteByte('\n')
		}
		body.WriteString(content)
		firstLine = false
		position, done = next, lineDone
	}

	return strings.TrimRight(body.String(), "\n"), position, done
}

// deindent removes one callout-body indentation level and reports whether it was present.
func deindent(line string) (string, bool) {
	if content, ok := strings.CutPrefix(line, "    "); ok {
		return content, true
	}
	if content, ok := strings.CutPrefix(line, "\t"); ok {
		return content, true
	}
	return line, false
}

// possibleFence cheaply rejects ordinary lines before invoking the Markdown fence parser.
func possibleFence(line string) bool {
	return strings.IndexByte(line, '`') >= 0 || strings.IndexByte(line, '~') >= 0
}

// sourceLine returns one line without its newline and the start position of the next line.
// A source ending in a newline exposes the same final empty line as strings.Split(source, "\n").
func sourceLine(source string, position int) (line string, next int, done bool) {
	if position > len(source) {
		return "", position, true
	}
	if newline := strings.IndexByte(source[position:], '\n'); newline >= 0 {
		end := position + newline
		return source[position:end], end + 1, false
	}
	return source[position:], len(source) + 1, true
}

// calloutKind normalizes and validates a supported callout kind.
func calloutKind(line string) string {
	if !strings.Contains(line, "!!! ") {
		return ""
	}
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "!!! ")
	if !ok {
		return ""
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	if end := strings.IndexAny(body, " \t\r\n"); end >= 0 {
		body = body[:end]
	}
	kind := strings.ToLower(body)
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
