package main

import (
	stdhtml "html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/markdownblock"
	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

// tabSection contains one parsed tab label and its de-indented Markdown body.
type tabSection struct {
	title string
	body  string
}

// transformTabs replaces top-level tab groups while keeping each panel body as
// host-rendered Markdown. This preserves nested Kumbuka syntax without giving the
// plugin direct access to the renderer.
func transformTabs(source string) []sdk.RenderPart {
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

		title, ok := parseTabTitle(lines[index])
		if !ok {
			plain = append(plain, lines[index])
			index++
			continue
		}

		sections := make([]tabSection, 0, 2)
		for ok {
			bodyLines, next := markdownblock.IndentedBody(lines, index+1)
			sections = append(sections, tabSection{title: title, body: strings.Join(bodyLines, "\n")})
			index = next
			if index >= len(lines) {
				break
			}
			title, ok = parseTabTitle(lines[index])
		}

		flushPlain()
		markdownblock.AppendText(&parts, "\n<div class=\"markdown-tabs\"><div class=\"markdown-tab-list\" role=\"tablist\">")
		for sectionIndex, section := range sections {
			class := "markdown-tab"
			selected := "false"
			if sectionIndex == 0 {
				class += " active"
				selected = "true"
			}
			markdownblock.AppendText(&parts, `<button type="button" class="`+class+`" role="tab" aria-selected="`+selected+`">`+stdhtml.EscapeString(section.title)+`</button>`)
		}
		markdownblock.AppendText(&parts, `</div><div class="markdown-tab-panels">`)

		for sectionIndex, section := range sections {
			class := "markdown-tab-panel"
			if sectionIndex != 0 {
				class += " markdown-tab-panel-hidden"
			}
			markdownblock.AppendText(&parts, `<div class="`+class+`" role="tabpanel">`)
			body := section.body
			parts = append(parts, sdk.RenderPart{Markdown: &body})
			markdownblock.AppendText(&parts, `</div>`)
		}
		markdownblock.AppendText(&parts, "</div></div>\n")
	}

	flushPlain()
	if len(parts) == 0 {
		return []sdk.RenderPart{{Text: source}}
	}

	return parts
}

// appendText appends literal output and coalesces adjacent text fragments.
// parseTabTitle parses a top-level declaration such as === "Linux".
func parseTabTitle(line string) (string, bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false
	}

	remaining, ok := strings.CutPrefix(strings.TrimSpace(line), "===")
	if !ok {
		return "", false
	}

	return markdownblock.ParseQuotedTitle(strings.TrimSpace(remaining))
}

// parseQuotedTitle parses one non-empty Go-style quoted title.
// indentedBody collects blank and four-space- or tab-indented body lines.
// stripBlockIndent removes one supported custom-block indentation level.
