// Package markdownblock provides shared helpers for first-party custom Markdown blocks.
package markdownblock

import (
	"strconv"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

// AppendText appends literal output while coalescing adjacent text fragments.
func AppendText(parts *[]sdk.RenderPart, text string) {
	if text == "" {
		return
	}
	if len(*parts) != 0 && (*parts)[len(*parts)-1].Markdown == nil {
		(*parts)[len(*parts)-1].Text += text
		return
	}
	*parts = append(*parts, sdk.RenderPart{Text: text})
}

// ParseQuotedTitle parses one non-empty Go-style double-quoted title.
func ParseQuotedTitle(value string) (string, bool) {
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false
	}

	title, err := strconv.Unquote(value)
	if err != nil || strings.TrimSpace(title) == "" {
		return "", false
	}
	return title, true
}

// IndentedBody collects blank and four-space- or tab-indented Markdown body lines.
func IndentedBody(lines []string, start int) ([]string, int) {
	body := make([]string, 0)
	index := start
	for index < len(lines) {
		if strings.TrimSpace(lines[index]) == "" {
			body = append(body, "")
			index++
			continue
		}

		line, ok := StripBlockIndent(lines[index])
		if !ok {
			break
		}
		body = append(body, line)
		index++
	}
	return body, index
}

// StripBlockIndent removes one supported custom-block indentation level.
func StripBlockIndent(line string) (string, bool) {
	if content, ok := strings.CutPrefix(line, "\t"); ok {
		return content, true
	}
	if content, ok := strings.CutPrefix(line, "    "); ok {
		return content, true
	}
	return "", false
}
