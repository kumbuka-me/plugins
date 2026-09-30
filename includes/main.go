package main

import (
	"fmt"
	"html"
	"maps"
	"strings"

	"github.com/kumbuka-me/plugins/internal/ascii"
	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const maxIncludeDepth = 5

// main runs the package entry point.
func main() {}

// init registers the Includes content preprocessor with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("includes", transform) }

// pageLoader returns authorized page Markdown for one canonical path.
type pageLoader func(string) (sdk.PageContent, error)

// transform expands Includes during the content-preprocess stage.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "content-preprocess" {
		return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: request.Source}}}
	}

	markdown, err := expandIncludes(request.Source, sdk.Pages().Content, nil, 0, localize.For(request.Locale))
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: markdown}}}
}

// expandIncludes resolves includes outside fenced code blocks.
func expandIncludes(source string, load pageLoader, seen map[string]bool, depth int, localizer sdk.Localizer) (string, error) {
	if depth > maxIncludeDepth {
		return "", fmt.Errorf("include expansion exceeds maximum depth of %d", maxIncludeDepth)
	}
	if seen == nil {
		seen = map[string]bool{}
	}
	if !strings.Contains(source, "{{include:") {
		return source, nil
	}

	var output strings.Builder
	output.Grow(len(source))
	fence := ""
	for position := 0; position <= len(source); {
		line, next, done := sourceLine(source, position)
		transformed := line

		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
		} else if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
		} else if strings.Contains(line, "{{include:") {
			var err error
			transformed, err = expandLine(line, load, seen, depth, localizer)
			if err != nil {
				return "", err
			}
		}

		output.WriteString(transformed)
		if done {
			break
		}
		output.WriteByte('\n')
		position = next
	}

	return output.String(), nil
}

// expandLine replaces every well-formed include macro in one source line.
func expandLine(line string, load pageLoader, seen map[string]bool, depth int, localizer sdk.Localizer) (string, error) {
	var output strings.Builder
	for {
		macro, ok := parseInclude(line)
		if !ok {
			output.WriteString(line)
			break
		}
		output.WriteString(line[:macro.start])
		included, err := expandInclude(macro.target, load, seen, depth, localizer)
		if err != nil {
			return "", err
		}
		output.WriteString(included)
		line = line[macro.end:]
	}
	return output.String(), nil
}

// includeMacro records the byte range and target of one parsed include expression.
type includeMacro struct {
	// start is the byte offset where the macro begins.
	start int
	// end is the byte offset immediately after the macro.
	end int
	// target is the requested page path with an optional heading fragment.
	target string
}

// parseInclude finds the next valid {{include:path}} macro.
func parseInclude(line string) (includeMacro, bool) {
	const opening = "{{include:"
	for offset := 0; offset < len(line); {
		found := strings.Index(line[offset:], opening)
		if found < 0 {
			return includeMacro{}, false
		}
		start := offset + found
		bodyStart := start + len(opening)
		close := strings.Index(line[bodyStart:], "}}")
		if close < 0 {
			return includeMacro{}, false
		}
		end := bodyStart + close
		target := strings.TrimSpace(line[bodyStart:end])
		if target != "" && !strings.ContainsAny(target, "{}\r\n") {
			return includeMacro{start: start, end: end + 2, target: target}, true
		}
		offset = end + 2
	}
	return includeMacro{}, false
}

// expandInclude resolves one whole-page or heading-section include recursively.
func expandInclude(target string, load pageLoader, seen map[string]bool, depth int, localizer sdk.Localizer) (string, error) {
	page, heading, _ := strings.Cut(strings.TrimSpace(target), "#")
	slug := strings.Trim(strings.TrimSpace(page), "/")
	heading = strings.TrimSpace(heading)
	if slug == "" {
		return "", fmt.Errorf("include requires a page path")
	}
	key := includeKey(slug, heading)
	if seen[key] {
		return "", fmt.Errorf("recursive page include %q", key)
	}

	content, err := load(slug)
	if err != nil {
		return "", fmt.Errorf("include page %q: %w", slug, err)
	}
	markdown, err := includedMarkdown(content.Markdown, slug, heading)
	if err != nil {
		return "", err
	}
	nextSeen := maps.Clone(seen)
	nextSeen[key] = true
	expanded, err := expandIncludes(markdown, load, nextSeen, depth+1, localizer)
	if err != nil {
		return "", fmt.Errorf("expand include %s: %w", slug, err)
	}
	return includeBreadcrumb(slug, heading, localizer) + expanded, nil
}

// includeBreadcrumb identifies the source of transcluded content using Kumbuka's page breadcrumb convention.
func includeBreadcrumb(slug, heading string, localizer sdk.Localizer) string {
	parts := []string{localizer.Text("common.pages"), slug}
	if heading != "" {
		parts = append(parts, heading)
	}
	for index := range parts {
		parts[index] = html.EscapeString(parts[index])
	}
	return `<p class="breadcrumbs include-breadcrumbs">` + html.EscapeString(localizer.Text("include.from")) + ` · ` + strings.Join(parts, " / ") + "</p>\n\n"
}

// includeKey returns the recursion key for a complete page or heading section.
func includeKey(slug, heading string) string {
	if heading == "" {
		return slug
	}
	return slug + "#" + headingID(heading)
}

// includedMarkdown selects one heading section when requested.
func includedMarkdown(source, slug, heading string) (string, error) {
	if heading == "" {
		return source, nil
	}
	section, err := markdownSection(source, heading)
	if err != nil {
		return "", fmt.Errorf("include section %s#%s: %w", slug, heading, err)
	}
	return section, nil
}

// markdownSection returns an ATX heading through the next sibling or ancestor heading.
func markdownSection(source, requested string) (string, error) {
	requestedID := headingID(requested)
	start := -1
	level := 0
	fence := ""

	for position := 0; position <= len(source); {
		lineStart := position
		line, next, done := sourceLine(source, position)

		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
		} else if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
		} else if headingLevel, title, ok := atxHeading(line); ok {
			if start < 0 && headingID(title) == requestedID {
				start = lineStart
				level = headingLevel
			} else if start >= 0 && headingLevel <= level {
				end := lineStart
				if end > start && source[end-1] == '\n' {
					end--
				}
				return source[start:end], nil
			}
		}

		if done {
			break
		}
		position = next
	}
	if start < 0 {
		return "", fmt.Errorf("heading %q not found", requested)
	}
	return source[start:], nil
}

// atxHeading parses one Markdown ATX heading.
func atxHeading(line string) (level int, title string, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || !strings.HasPrefix(trimmed, "#") {
		return 0, "", false
	}
	for level < len(trimmed) && level < 6 && trimmed[level] == '#' {
		level++
	}
	if level < len(trimmed) && trimmed[level] != ' ' && trimmed[level] != '\t' {
		return 0, "", false
	}
	title = strings.TrimSpace(trimmed[level:])
	closing := strings.TrimRight(title, "#")
	if closing == "" || strings.HasSuffix(closing, " ") || strings.HasSuffix(closing, "\t") {
		title = strings.TrimSpace(closing)
	}
	return level, title, true
}

// headingID mirrors Kumbuka's stable ASCII heading-anchor normalization.
func headingID(value string) string {
	value = strings.TrimSpace(value)
	var output strings.Builder
	output.Grow(len(value))
	separator := false
	for index := 0; index < len(value); index++ {
		character := value[index]
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		if ascii.IsAlphaNumeric(character) || character == '_' || character == '-' {
			if separator && output.Len() > 0 {
				output.WriteByte('-')
			}
			separator = false
			output.WriteByte(character)
		} else {
			separator = true
		}
	}
	return strings.Trim(output.String(), "-")
}

// sourceLine returns one line without its newline and the start position of the next line.
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
