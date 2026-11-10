// Package codepresentation renders source lines, highlights, and annotations for first-party plugins.
package codepresentation

import (
	"fmt"
	"html"
	"strconv"
	"strings"
)

// Range identifies an inclusive one-based source-line interval.
type Range struct {
	// Start is the first included source line.
	Start int
	// End is the last included source line.
	End int
}

// Note is a plain-text annotation attached to an inclusive source-line range.
type Note struct {
	// Start is the first referenced source line.
	Start int
	// End is the last referenced source line.
	End int
	// Text is the annotation's untrusted plain-text content.
	Text string
}

// Appearance selects the source presentation's validated theme-safe styles.
type Appearance struct {
	// ReferencePosition selects the left or right marker gutter.
	ReferencePosition string
	// ReferenceColor selects the theme-safe color name.
	ReferenceColor string
	// HighlightReferences controls background shading for noted lines.
	HighlightReferences bool
	// ShowLineNumbers controls the source-line number column.
	ShowLineNumbers bool
}

// Labels provides localized strings used by the annotation legend and marker titles.
type Labels struct {
	// Line is the singular label for a source line.
	Line string
	// Lines is the plural label for source lines.
	Lines string
	// Annotation returns the marker's accessible title.
	Annotation func(number int) string
}

// Block contains one validated code presentation, independent of its source provider.
type Block struct {
	// ClassPrefix is the trusted CSS class namespace, selected by the caller.
	ClassPrefix string
	// Source is the plain-text code to render.
	Source string
	// StartLine is the original one-based number of Source's first line.
	StartLine int
	// Notes are validated annotations within the selected source range.
	Notes []Note
	// Highlights are optional independent line ranges receiving background shading.
	Highlights []Range
	// Appearance contains the validated visual configuration.
	Appearance Appearance
	// Labels contains localizable annotation text.
	Labels Labels
}

// ValidPosition reports whether value identifies a supported note gutter.
func ValidPosition(value string) bool { return value == "left" || value == "right" }

// ValidColor reports whether value identifies a supported presentation color.
func ValidColor(value string) bool {
	switch value {
	case "accent", "blue", "green", "yellow", "orange", "red", "purple", "gray":
		return true
	default:
		return false
	}
}

// ParseRange parses one inclusive one-based line or range bounded by maximum.
func ParseRange(value string, maximum int) (Range, bool) {
	first, last, hasRange := strings.Cut(value, "-")
	if !hasRange {
		last = first
	}
	start, err := strconv.Atoi(first)
	if err != nil {
		return Range{}, false
	}
	end, err := strconv.Atoi(last)
	if err != nil || start < 1 || end < start || end > maximum {
		return Range{}, false
	}
	return Range{Start: start, End: end}, true
}

// ParseNote parses an annotated source range and a bounded nonempty description.
func ParseNote(value string, maxLine, maxBytes int) (Note, bool) {
	selection, description, ok := strings.Cut(value, ":")
	if !ok || strings.TrimSpace(description) == "" || len(description) > maxBytes {
		return Note{}, false
	}
	selected, ok := ParseRange(selection, maxLine)
	if !ok {
		return Note{}, false
	}
	return Note{Start: selected.Start, End: selected.End, Text: description}, true
}

// ParseHighlights parses comma-separated inclusive source lines and ranges.
func ParseHighlights(value string, maxLine, maxRanges int) ([]Range, bool) {
	if value == "" {
		return nil, false
	}
	parts := strings.Split(value, ",")
	if len(parts) > maxRanges {
		return nil, false
	}
	result := make([]Range, 0, len(parts))
	for _, part := range parts {
		selected, ok := ParseRange(strings.TrimSpace(part), maxLine)
		if !ok {
			return nil, false
		}
		result = append(result, selected)
	}
	return result, true
}

// NotesWithin checks every note against the displayed range and annotation size limit.
func NotesWithin(notes []Note, start, count, maxBytes int) bool {
	end := start + count - 1
	for _, note := range notes {
		if note.Start < start || note.End < note.Start || note.End > end || len(note.Text) > maxBytes {
			return false
		}
	}
	return true
}

// RangesWithin checks every highlight range against the displayed source lines.
func RangesWithin(ranges []Range, start, count int) bool {
	end := start + count - 1
	for _, selection := range ranges {
		if selection.Start < start || selection.End < selection.Start || selection.End > end {
			return false
		}
	}
	return true
}

// WriteCode writes escaped source lines and accessible annotation markers into an HTML builder.
func WriteCode(output *strings.Builder, block Block) {
	markers := make(map[int][]int, len(block.Notes))
	shaded := make(map[int]bool)
	for index, note := range block.Notes {
		markers[note.Start] = append(markers[note.Start], index+1)
		if block.Appearance.HighlightReferences {
			for line := note.Start; line <= note.End; line++ {
				shaded[line] = true
			}
		}
	}
	for _, selected := range block.Highlights {
		for line := selected.Start; line <= selected.End; line++ {
			shaded[line] = true
		}
	}

	prefix := block.ClassPrefix
	output.WriteString(`<pre class="` + prefix + `-code"><code>`)
	for index, line := range strings.Split(block.Source, "\n") {
		number := block.StartLine + index
		className := prefix + "-line"
		if shaded[number] {
			className += " " + prefix + "-line-annotated"
		}
		output.WriteString(`<span class="` + className + `">`)
		output.WriteString(`<span class="` + prefix + `-number">` + strconv.Itoa(number) + `</span>`)
		if block.Appearance.ReferencePosition == "left" {
			writeMarkers(output, prefix, markers[number], block.Labels)
		}
		output.WriteString(`<span class="` + prefix + `-source">` + html.EscapeString(line) + `</span>`)
		if block.Appearance.ReferencePosition == "right" {
			writeMarkers(output, prefix, markers[number], block.Labels)
		}
		output.WriteString(`</span>`)
	}
	output.WriteString(`</code></pre>`)
}

// writeMarkers writes the annotation badges for a single source line.
func writeMarkers(output *strings.Builder, prefix string, numbers []int, labels Labels) {
	output.WriteString(`<span class="` + prefix + `-gutter">`)
	for _, number := range numbers {
		title := fmt.Sprintf("Annotation %d", number)
		if labels.Annotation != nil {
			title = labels.Annotation(number)
		}
		fmt.Fprintf(output, `<span class="%s-marker" title="%s">%d</span>`, prefix, html.EscapeString(title), number)
	}
	output.WriteString(`</span>`)
}

// WriteNotes appends an escaped numbered note legend below the source block.
func WriteNotes(output *strings.Builder, block Block) {
	if len(block.Notes) == 0 {
		return
	}
	prefix := block.ClassPrefix
	output.WriteString(`<div class="` + prefix + `-notes"><ol class="` + prefix + `-note-list">`)
	for index, note := range block.Notes {
		className := prefix + "-note"
		if index > 0 {
			className += " " + prefix + "-note-following"
		}
		output.WriteString(`<li class="` + className + `"><span class="` + prefix + `-note-marker">`)
		output.WriteString(strconv.Itoa(index + 1))
		output.WriteString(`</span><span><strong>`)
		if note.Start == note.End {
			output.WriteString(html.EscapeString(block.Labels.Line) + " " + strconv.Itoa(note.Start))
		} else {
			output.WriteString(html.EscapeString(block.Labels.Lines) + " " + strconv.Itoa(note.Start) + "–" + strconv.Itoa(note.End))
		}
		output.WriteString(`:</strong> ` + html.EscapeString(note.Text) + `</span></li>`)
	}
	output.WriteString(`</ol></div>`)
}
