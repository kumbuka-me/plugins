package main

import (
	stdhtml "html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/markdownblock"
	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

// transformDetails replaces top-level ??? blocks while keeping each body as
// host-rendered Markdown so nested Kumbuka syntax follows the normal pipeline.
func transformDetails(source string) []sdk.RenderPart {
	lines := strings.Split(source, "\n")
	plain := make([]string, 0, len(lines))
	parts := make([]sdk.RenderPart, 0)

	flushPlain := func() {
		if len(plain) == 0 {
			return
		}
		markdownblock.AppendText(&parts, strings.Join(plain, "\n"))
		plain = plain[:0]
	}

	for index := 0; index < len(lines); {
		if marker := pluginmarkdown.Fence(lines[index]); marker != "" {
			index = pluginmarkdown.AppendFence(lines, index, marker, &plain)
			continue
		}

		title, open, ok := parseDetailsTitle(lines[index])
		if !ok {
			plain = append(plain, lines[index])
			index++
			continue
		}

		bodyLines, next := markdownblock.IndentedBody(lines, index+1)
		flushPlain()

		openAttribute := ""
		if open {
			openAttribute = " open"
		}
		markdownblock.AppendText(&parts, "\n<details class=\"markdown-details\""+openAttribute+`><summary>`+stdhtml.EscapeString(title)+`</summary><div class="markdown-details-body">`)
		body := strings.Join(bodyLines, "\n")
		parts = append(parts, sdk.RenderPart{Markdown: &body})
		markdownblock.AppendText(&parts, "</div></details>\n")
		index = next
	}

	flushPlain()
	if len(parts) == 0 {
		return []sdk.RenderPart{{Text: source}}
	}

	return parts
}

// appendText appends literal output and coalesces adjacent text fragments.
// parseDetailsTitle parses ??? and ???+ declarations.
func parseDetailsTitle(line string) (title string, open bool, ok bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false, false
	}

	trimmed := strings.TrimSpace(line)
	remaining, open := strings.CutPrefix(trimmed, "???+")
	if !open {
		var found bool
		remaining, found = strings.CutPrefix(trimmed, "???")
		if !found {
			return "", false, false
		}
	}

	title, ok = markdownblock.ParseQuotedTitle(strings.TrimSpace(remaining))
	return title, open, ok
}

// parseQuotedTitle parses one non-empty Go-style quoted title.
// indentedBody collects blank and four-space- or tab-indented body lines.
// stripBlockIndent removes one supported custom-block indentation level.
