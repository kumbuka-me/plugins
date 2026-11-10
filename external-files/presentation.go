package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/kumbuka-me/plugins/internal/codepresentation"
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

// validReferencePosition checks the shared code presentation gutter choices.
func validReferencePosition(value string) bool { return codepresentation.ValidPosition(value) }

// validReferenceColor checks the shared code presentation color choices.
func validReferenceColor(value string) bool { return codepresentation.ValidColor(value) }

// message returns one escaped user-facing external-file error box.
func message(text string) sdk.Result {
	return sdk.Text(`<div class="external-file-error">` + html.EscapeString(text) + `</div>`)
}

// render loads settings and source configuration, fetches the file, validates notes, and renders escaped text.
func render(value options, resources resourceReader, settings settingsReader, httpDo httpDoer, localizer sdk.Localizer) sdk.Result {
	if value.Invalid || value.Source == "" || value.Path == "" || len(value.Notes) > maxAnnotations {
		return message(localizer.Text("external.invalid"))
	}

	source, err := loadSource(value.Source, resources)
	if err != nil {
		return message(localizer.Text("external.unavailable_admin"))
	}
	content, err := cachedFile(value.Source, source, value.Path, loadCacheTTL(settings), time.Now().UTC(), httpDo)
	if err != nil {
		return message(localizer.Text("external.unavailable_admin"))
	}
	file, err := selectLines(content, value.Start, value.End)
	if err != nil {
		return message(localizer.Text("external.unavailable_range"))
	}
	if !annotationsWithinSelection(value.Notes, file) {
		return message(localizer.Text("external.annotation_range"))
	}

	appearance := applyPresentationOverrides(loadPresentation(settings), value)
	return sdk.Text(renderExternalFile(value, source, file, appearance, localizer))
}

// annotationsWithinSelection checks notes against the selected original lines.
func annotationsWithinSelection(notes []annotation, file selectedFile) bool {
	return codepresentation.NotesWithin(notes, file.Start, strings.Count(file.Content, "\n")+1, maxAnnotationBytes)
}

// renderExternalFile builds the provider header, shared source rows, and note legend.
func renderExternalFile(value options, source source, file selectedFile, appearance presentation, localizer sdk.Localizer) string {
	var output strings.Builder
	lineNumbersClass := ""
	if !appearance.ShowLineNumbers {
		lineNumbersClass = " external-file-hide-line-numbers"
	}
	fmt.Fprintf(&output, `<div class="external-file external-file-reference-%s external-file-color-%s%s">`, appearance.ReferencePosition, appearance.ReferenceColor, lineNumbersClass)
	writeHeader(&output, value.Path, source, appearance)
	block := codepresentation.Block{
		ClassPrefix: "external-file",
		Source:      file.Content,
		StartLine:   file.Start,
		Notes:       value.Notes,
		Appearance: codepresentation.Appearance{
			ReferencePosition:   appearance.ReferencePosition,
			ReferenceColor:      appearance.ReferenceColor,
			HighlightReferences: appearance.HighlightReferences,
			ShowLineNumbers:     appearance.ShowLineNumbers,
		},
		Labels: codepresentation.Labels{
			Line:       localizer.Text("external.line"),
			Lines:      localizer.Text("external.lines"),
			Annotation: func(number int) string { return localizer.Textf("external.annotation", number) },
		},
	}
	codepresentation.WriteCode(&output, block)
	codepresentation.WriteNotes(&output, block)
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

// providerLabel returns the display label for one validated repository provider.
func providerLabel(provider string) string {
	if provider == "gitlab" {
		return "GitLab"
	}
	return "GitHub"
}
