package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

const appearanceSettingPrefix = "appearance."

// presentation contains resolved global defaults and per-embed display overrides.
type presentation struct {
	// ReferencePosition selects the left or right annotation gutter.
	ReferencePosition string
	// ReferenceColor selects the theme-safe annotation color class.
	ReferenceColor string
	// HighlightReferences controls subtle background highlighting on annotated lines.
	HighlightReferences bool
	// ShowLineNumbers controls the conventional left source-line number gutter.
	ShowLineNumbers bool
	// ShowProvider controls the GitHub or GitLab badge in the header.
	ShowProvider bool
	// ShowBranch controls the configured branch, tag, or commit badge in the header.
	ShowBranch bool
}

// defaultPresentation returns the plugin-owned presentation defaults used before an administrator saves settings.
func defaultPresentation() presentation {
	return presentation{
		ReferencePosition:   "right",
		ReferenceColor:      "accent",
		HighlightReferences: true,
		ShowLineNumbers:     true,
		ShowProvider:        true,
		ShowBranch:          true,
	}
}

// loadPresentation reads typed plugin settings and falls back to manifest-equivalent defaults on capability errors.
func loadPresentation(read settingsReader) presentation {
	result := defaultPresentation()
	if read == nil {
		return result
	}

	if value, ok := readSetting(read, "reference_position"); ok && validReferencePosition(value) {
		result.ReferencePosition = value
	}
	if value, ok := readSetting(read, "reference_color"); ok && validReferenceColor(value) {
		result.ReferenceColor = value
	}
	result.HighlightReferences = readBoolSetting(read, "highlight_referenced_lines", result.HighlightReferences)
	result.ShowLineNumbers = readBoolSetting(read, "show_line_numbers", result.ShowLineNumbers)
	result.ShowProvider = readBoolSetting(read, "show_provider", result.ShowProvider)
	result.ShowBranch = readBoolSetting(read, "show_branch", result.ShowBranch)
	return result
}

// readSetting reads one typed appearance setting as text.
func readSetting(read settingsReader, key string) (string, bool) {
	value, err := read(appearanceSettingPrefix + key)
	if err != nil || !value.Found {
		return "", false
	}
	return string(value.Value), true
}

// readBoolSetting reads one typed appearance boolean or returns fallback for malformed or unavailable values.
func readBoolSetting(read settingsReader, key string, fallback bool) bool {
	value, ok := readSetting(read, key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

// applyPresentationOverrides applies optional macro-local display values over plugin defaults.
func applyPresentationOverrides(base presentation, value options) presentation {
	if value.ReferencePosition != "" {
		base.ReferencePosition = value.ReferencePosition
	}
	if value.ReferenceColor != "" {
		base.ReferenceColor = value.ReferenceColor
	}
	if value.HighlightReferences.Set {
		base.HighlightReferences = value.HighlightReferences.Value
	}
	if value.ShowLineNumbers.Set {
		base.ShowLineNumbers = value.ShowLineNumbers.Value
	}
	if value.ShowProvider.Set {
		base.ShowProvider = value.ShowProvider.Value
	}
	if value.ShowBranch.Set {
		base.ShowBranch = value.ShowBranch.Value
	}
	return base
}

// validReferencePosition reports whether value is a supported annotation-gutter side.
func validReferencePosition(value string) bool {
	return value == "left" || value == "right"
}

// validReferenceColor reports whether value is a supported theme-safe annotation color.
func validReferenceColor(value string) bool {
	switch value {
	case "accent", "blue", "green", "yellow", "orange", "red", "purple", "gray":
		return true
	default:
		return false
	}
}

// message returns one escaped user-facing external-file error box.
func message(text string) sdk.Result {
	return sdk.Text(`<div class="external-file-error">` + html.EscapeString(text) + `</div>`)
}

// render loads settings and source configuration, fetches the file, validates notes, and renders escaped text.
func render(value options, resources resourceReader, settings settingsReader, httpDo httpDoer) sdk.Result {
	if value.Invalid || value.Source == "" || value.Path == "" || len(value.Notes) > maxAnnotations {
		return message("Invalid external file. Use source and path, optional lines, notes, and supported presentation overrides.")
	}

	source, err := loadSource(value.Source, resources)
	if err != nil {
		return message("External file unavailable. Ask an administrator to check the configured source and provider access.")
	}
	content, err := cachedFile(value.Source, source, value.Path, loadCacheTTL(settings), time.Now().UTC(), httpDo)
	if err != nil {
		return message("External file unavailable. Ask an administrator to check the configured source and provider access.")
	}
	file, err := selectLines(content, value.Start, value.End)
	if err != nil {
		return message("External file unavailable. Check the requested line range.")
	}
	if !annotationsWithinSelection(value.Notes, file) {
		return message("An annotation refers to a line outside the displayed file range.")
	}

	appearance := applyPresentationOverrides(loadPresentation(settings), value)
	return sdk.Text(renderExternalFile(value, source, file, appearance))
}

// annotationsWithinSelection reports whether every note range is contained by the displayed source.
func annotationsWithinSelection(notes []annotation, file selectedFile) bool {
	lineCount := len(strings.Split(file.Content, "\n"))
	lastLine := file.Start + lineCount - 1
	for _, note := range notes {
		if note.Start < file.Start || note.End < note.Start || note.End > lastLine || len(note.Text) > maxAnnotationBytes {
			return false
		}
	}
	return true
}

// renderExternalFile builds the provider header, source rows, gutters, and note legend.
func renderExternalFile(value options, source source, file selectedFile, appearance presentation) string {
	var output strings.Builder
	lineNumbersClass := ""
	if !appearance.ShowLineNumbers {
		lineNumbersClass = " external-file-hide-line-numbers"
	}
	fmt.Fprintf(
		&output,
		`<div class="external-file external-file-reference-%s external-file-color-%s%s">`,
		appearance.ReferencePosition,
		appearance.ReferenceColor,
		lineNumbersClass,
	)
	writeHeader(&output, value.Path, source, appearance)
	writeCode(&output, file, value.Notes, appearance)
	writeNotes(&output, value.Notes)
	output.WriteString(`</div>`)
	return output.String()
}

// writeHeader appends the compact provider, repository, path, and revision header.
func writeHeader(output *strings.Builder, path string, source source, appearance presentation) {
	output.WriteString(`<div class="external-file-header"><div class="external-file-location">`)
	if appearance.ShowProvider {
		output.WriteString(`<span class="external-file-provider">`)
		output.WriteString(html.EscapeString(providerLabel(source.Provider)))
		output.WriteString(`</span>`)
	}
	output.WriteString(`<strong class="external-file-repository">`)
	output.WriteString(html.EscapeString(source.Repository))
	output.WriteString(`</strong><span class="external-file-path">/ `)
	output.WriteString(html.EscapeString(path))
	output.WriteString(`</span></div>`)
	if appearance.ShowBranch {
		output.WriteString(`<span class="external-file-ref">`)
		output.WriteString(html.EscapeString(source.Ref))
		output.WriteString(`</span>`)
	}
	output.WriteString(`</div>`)
}

// writeCode appends escaped source lines with a dedicated configurable annotation gutter.
func writeCode(output *strings.Builder, file selectedFile, notes []annotation, appearance presentation) {
	markers := make(map[int][]int)
	annotated := make(map[int]bool)
	for index, note := range notes {
		markers[note.Start] = append(markers[note.Start], index+1)
		for line := note.Start; line <= note.End; line++ {
			annotated[line] = true
		}
	}

	output.WriteString(`<pre class="external-file-code"><code>`)
	lines := strings.Split(file.Content, "\n")
	for index, line := range lines {
		number := file.Start + index
		lineMarkers := markers[number]
		classes := "external-file-line"
		if appearance.HighlightReferences && annotated[number] {
			classes += " external-file-line-annotated"
		}
		output.WriteString(`<span class="` + classes + `">`)
		output.WriteString(`<span class="external-file-number">` + strconv.Itoa(number) + `</span>`)
		if appearance.ReferencePosition == "left" {
			writeMarkerGutter(output, lineMarkers)
		}
		output.WriteString(`<span class="external-file-source">` + html.EscapeString(line) + `</span>`)
		if appearance.ReferencePosition == "right" {
			writeMarkerGutter(output, lineMarkers)
		}
		output.WriteString(`</span>`)
	}
	output.WriteString(`</code></pre>`)
}

// writeMarkerGutter appends one fixed source-line annotation gutter.
func writeMarkerGutter(output *strings.Builder, markers []int) {
	output.WriteString(`<span class="external-file-gutter">`)
	for _, marker := range markers {
		fmt.Fprintf(output, `<span class="external-file-marker" title="Annotation %d">%d</span>`, marker, marker)
	}
	output.WriteString(`</span>`)
}

// writeNotes appends the annotation legend below the source block.
func writeNotes(output *strings.Builder, notes []annotation) {
	if len(notes) == 0 {
		return
	}

	output.WriteString(`<div class="external-file-notes"><ol class="external-file-note-list">`)
	for index, note := range notes {
		className := "external-file-note"
		if index > 0 {
			className += " external-file-note-following"
		}
		output.WriteString(`<li class="` + className + `"><span class="external-file-note-marker">`)
		output.WriteString(strconv.Itoa(index + 1))
		output.WriteString(`</span><span><strong>`)
		if note.Start == note.End {
			output.WriteString("Line ")
			output.WriteString(strconv.Itoa(note.Start))
		} else {
			output.WriteString("Lines ")
			output.WriteString(strconv.Itoa(note.Start))
			output.WriteString("–")
			output.WriteString(strconv.Itoa(note.End))
		}
		output.WriteString(`:</strong> `)
		output.WriteString(html.EscapeString(note.Text))
		output.WriteString(`</span></li>`)
	}
	output.WriteString(`</ol></div>`)
}

// providerLabel returns the display label for one validated repository provider.
func providerLabel(provider string) string {
	if provider == "gitlab" {
		return "GitLab"
	}
	return "GitHub"
}
