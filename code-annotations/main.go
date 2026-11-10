// Package main implements opt-in annotations for fenced Markdown source code.
package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/kumbuka-me/plugins/internal/codepresentation"
	"github.com/kumbuka-me/plugins/internal/macroargs"
	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const (
	maxCodeBytes       = 128 << 10
	maxCodeLines       = 10000
	maxAnnotations     = 32
	maxAnnotationBytes = 2048
	maxHighlights      = 64
	maxDirectiveBytes  = 16384
	maxMacroBytes      = 96 << 10
)

// options contains one explicit code presentation directive.
type options struct {
	// Language selects a display label when the block comes from a macro.
	Language string
	// Source contains the embedded code when the block comes from a macro.
	Source string
	// Invalid identifies a malformed matching macro.
	Invalid bool
	// Notes attach bounded plain-text comments to source ranges.
	Notes []codepresentation.Note
	// Highlights are independent background-highlight ranges.
	Highlights []codepresentation.Range
	// Position selects the note marker gutter.
	Position string
	// Color selects the marker and highlight color.
	Color string
	// HighlightReferences shades lines with annotations when true.
	HighlightReferences bool
	// LineNumbers enables the numbered line gutter.
	LineNumbers bool
}

// main is the WASI plugin entry point.
func main() {}

// init registers the opt-in Markdown preprocessor.
func init() {
	sdk.RegisterModule("code-annotations", transform)
	sdk.RegisterLocalizedMacro("annotated-code", parseMacro, func(invocation sdk.LocalizedMacro[options]) (sdk.Result, error) {
		value := invocation.Value
		if value.Invalid {
			return sdk.Text(`<div class="code-annotation-error">Invalid annotated-code invocation</div>`), nil
		}
		output, err := renderBlock(value.Language, value.Source, value)
		if err != nil {
			return sdk.Text(`<div class="code-annotation-error">` + html.EscapeString(err.Error()) + `</div>`), nil
		}
		return sdk.Text(output), nil
	})
}

// transform converts only closed fenced blocks followed by an explicit {code ...} directive.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" || request.Module != "code-annotations" {
		return sdk.RenderResult{Error: "unsupported code-annotations render request"}
	}
	if !strings.Contains(request.Source, "{code") {
		return sdk.Text(request.Source)
	}
	return sdk.Text(annotateFences(request.Source))
}

// annotateFences scans complete Markdown fences so directives inside ordinary code remain literal.
func annotateFences(source string) string {
	lines := strings.SplitAfter(source, "\n")
	var output strings.Builder
	for index := 0; index < len(lines); {
		opening := lineText(lines[index])
		marker := pluginmarkdown.Fence(opening)
		if marker == "" {
			output.WriteString(lines[index])
			index++
			continue
		}

		closing := index + 1
		for closing < len(lines) && !pluginmarkdown.Closes(lineText(lines[closing]), marker) {
			closing++
		}
		if closing >= len(lines) {
			output.WriteString(strings.Join(lines[index:], ""))
			break
		}

		following := closing + 1
		if following >= len(lines) || !isDirective(lineText(lines[following])) {
			output.WriteString(strings.Join(lines[index:following], ""))
			index = following
			continue
		}

		language := strings.TrimSpace(strings.TrimPrefix(strings.TrimLeft(opening, " \t"), marker))
		content := strings.Join(lines[index+1:closing], "")
		content = strings.TrimSuffix(strings.TrimSuffix(content, "\n"), "\r")
		rendered, err := renderAnnotated(language, content, strings.TrimSpace(lineText(lines[following])))
		if err != nil {
			output.WriteString(`<div class="code-annotation-error">` + html.EscapeString(err.Error()) + `</div>`)
		} else {
			output.WriteString(rendered)
		}
		output.WriteByte('\n')
		index = following + 1
	}
	return output.String()
}

// lineText returns a source line without its CRLF or LF delimiter.
func lineText(line string) string { return strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r") }

// isDirective reports whether a line has the explicit code-annotation directive prefix.
func isDirective(line string) bool {
	value := strings.TrimSpace(line)
	return value == "{code}" || strings.HasPrefix(value, "{code ")
}

// renderAnnotated validates code and directive options before generating escaped HTML.
func renderAnnotated(language, source, directive string) (string, error) {
	value, err := parseDirective(directive)
	if err != nil {
		return "", err
	}
	return renderBlock(language, source, value)
}

// renderBlock renders a validated opt-in code block regardless of its Markdown representation.
func renderBlock(language, source string, value options) (string, error) {
	if language == "" || strings.ContainsAny(language, " \t\r\n<>&\"'") || len(language) > 40 {
		return "", fmt.Errorf("code annotations: expected one fenced language name")
	}
	if len(source) > maxCodeBytes || strings.Count(source, "\n")+1 > maxCodeLines {
		return "", fmt.Errorf("code annotations: source exceeds the size or line limit")
	}
	count := strings.Count(source, "\n") + 1
	if !codepresentation.NotesWithin(value.Notes, 1, count, maxAnnotationBytes) || !codepresentation.RangesWithin(value.Highlights, 1, count) {
		return "", fmt.Errorf("code annotations: referenced lines are outside the code block")
	}
	appearance := codepresentation.Appearance{
		ReferencePosition:   value.Position,
		ReferenceColor:      value.Color,
		HighlightReferences: value.HighlightReferences,
		ShowLineNumbers:     value.LineNumbers,
	}
	block := codepresentation.Block{
		ClassPrefix: "code-annotation", Source: source, StartLine: 1,
		Notes: value.Notes, Highlights: value.Highlights, Appearance: appearance,
		Labels: codepresentation.Labels{Line: "Line", Lines: "Lines"},
	}
	var output strings.Builder
	fmt.Fprintf(&output, `<div class="code-annotation code-annotation-reference-%s code-annotation-color-%s`, value.Position, value.Color)
	if !value.LineNumbers {
		output.WriteString(` code-annotation-hide-line-numbers`)
	}
	output.WriteString(`"><div class="code-annotation-header">` + html.EscapeString(language) + `</div>`)
	codepresentation.WriteCode(&output, block)
	codepresentation.WriteNotes(&output, block)
	output.WriteString(`</div>`)
	return output.String(), nil
}

// parseMacro recognizes opt-in macro blocks used by Kumbuka's declarative visual editor.
func parseMacro(line string) (options, bool) {
	body, ok := strings.CutPrefix(strings.TrimSpace(line), "{{annotated-code ")
	if !ok {
		return options{}, false
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok {
		return options{}, false
	}
	invalid := options{Invalid: true}
	if len(body) > maxMacroBytes {
		return invalid, true
	}
	value, err := parseOptions(strings.TrimSpace(body), true)
	if err != nil || value.Language == "" || value.Source == "" {
		return invalid, true
	}
	return value, true
}

// parseDirective reads strictly quoted options and rejects unsupported or repeated scalar attributes.
func parseDirective(directive string) (options, error) {
	if len(directive) > maxDirectiveBytes || !strings.HasPrefix(directive, "{code") || !strings.HasSuffix(directive, "}") {
		return options{}, fmt.Errorf("code annotations: invalid directive")
	}
	body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(directive, "{code"), "}"))
	return parseOptions(body, false)
}

// parseOptions decodes shared note, range, and presentation settings for both opt-in syntaxes.
func parseOptions(body string, allowSource bool) (options, error) {
	value := options{Position: "right", Color: "accent", HighlightReferences: true, LineNumbers: true}
	seen := make(map[string]bool)
	for body != "" {
		name, arg, remaining, ok := macroargs.NextQuoted(body)
		if !ok || (seen[name] && name != "note") {
			return options{}, fmt.Errorf("code annotations: invalid or repeated option")
		}
		seen[name] = true
		body = strings.TrimSpace(remaining)
		switch name {
		case "language":
			if !allowSource {
				return options{}, fmt.Errorf("code annotations: language is defined by the fence")
			}
			value.Language = arg
		case "source":
			if !allowSource {
				return options{}, fmt.Errorf("code annotations: source is defined by the fence")
			}
			value.Source = arg
		case "note":
			note, valid := codepresentation.ParseNote(arg, maxCodeLines, maxAnnotationBytes)
			if !valid || len(value.Notes) >= maxAnnotations {
				return options{}, fmt.Errorf("code annotations: invalid or excessive note")
			}
			value.Notes = append(value.Notes, note)
		case "highlight":
			ranges, valid := codepresentation.ParseHighlights(arg, maxCodeLines, maxHighlights)
			if !valid {
				return options{}, fmt.Errorf("code annotations: invalid highlight range")
			}
			value.Highlights = ranges
		case "reference-position":
			if !codepresentation.ValidPosition(arg) {
				return options{}, fmt.Errorf("code annotations: invalid reference position")
			}
			value.Position = arg
		case "reference-color":
			if !codepresentation.ValidColor(arg) {
				return options{}, fmt.Errorf("code annotations: invalid reference color")
			}
			value.Color = arg
		case "highlight-references":
			parsed, valid := parseBool(arg)
			if !valid {
				return options{}, fmt.Errorf("code annotations: invalid highlight-references value")
			}
			value.HighlightReferences = parsed
		case "line-numbers":
			parsed, valid := parseBool(arg)
			if !valid {
				return options{}, fmt.Errorf("code annotations: invalid line-numbers value")
			}
			value.LineNumbers = parsed
		default:
			return options{}, fmt.Errorf("code annotations: unknown option %q", name)
		}
	}
	return value, nil
}

// parseBool accepts only explicit lowercase true or false values.
func parseBool(value string) (bool, bool) {
	return value == "true", value == "true" || value == "false"
}
