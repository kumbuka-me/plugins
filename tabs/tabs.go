package main

import (
	stdhtml "html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/renderparts"
	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

// tabSection contains one parsed tab label and its de-indented Markdown body.
type tabSection struct {
	// title is the human-readable label shown by the tab button.
	title string
	// body is the de-indented Markdown rendered inside the tab panel.
	body string
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
		renderparts.AppendText(&parts, strings.Join(plain, "\n"))
		plain = plain[:0]
	}

	for index := 0; index < len(lines); {
		if marker := pluginmarkdown.Fence(lines[index]); marker != "" {
			index = pluginmarkdown.AppendFence(lines, index, marker, &plain)
			continue
		}

		sections, next, ok := parseTabGroup(lines, index)
		if !ok {
			plain = append(plain, lines[index])
			index++
			continue
		}

		flushPlain()
		appendTabGroup(&parts, sections)
		index = next
	}

	flushPlain()
	if len(parts) == 0 {
		return []sdk.RenderPart{{Text: source}}
	}
	return parts
}

// parseTabGroup consumes consecutive top-level tab declarations and their bodies.
func parseTabGroup(lines []string, start int) ([]tabSection, int, bool) {
	title, ok := parseTabTitle(lines[start])
	if !ok {
		return nil, start, false
	}

	sections := make([]tabSection, 0, 2)
	index := start
	for {
		bodyLines, next := pluginmarkdown.IndentedBody(lines, index+1)
		sections = append(sections, tabSection{title: title, body: strings.Join(bodyLines, "\n")})
		index = next
		if index >= len(lines) {
			break
		}
		title, ok = parseTabTitle(lines[index])
		if !ok {
			break
		}
	}
	return sections, index, true
}

// appendTabGroup appends the host-rendered controls and Markdown panels for one tab group.
func appendTabGroup(parts *[]sdk.RenderPart, sections []tabSection) {
	renderparts.AppendText(parts, "\n<div class=\"markdown-tabs\"><div class=\"markdown-tab-list\" role=\"tablist\">")
	appendTabButtons(parts, sections)
	renderparts.AppendText(parts, `</div><div class="markdown-tab-panels">`)
	appendTabPanels(parts, sections)
	renderparts.AppendText(parts, "</div></div>\n")
}

// appendTabButtons appends accessible tab buttons with the first tab selected.
func appendTabButtons(parts *[]sdk.RenderPart, sections []tabSection) {
	for index, section := range sections {
		class := "markdown-tab"
		selected := "false"
		if index == 0 {
			class += " active"
			selected = "true"
		}
		renderparts.AppendText(parts, `<button type="button" class="`+class+`" role="tab" aria-selected="`+selected+`">`+stdhtml.EscapeString(section.title)+`</button>`)
	}
}

// appendTabPanels appends each tab body as host-rendered Markdown.
func appendTabPanels(parts *[]sdk.RenderPart, sections []tabSection) {
	for index, section := range sections {
		class := "markdown-tab-panel"
		if index != 0 {
			class += " markdown-tab-panel-hidden"
		}
		renderparts.AppendText(parts, `<div class="`+class+`" role="tabpanel">`)
		body := section.body
		*parts = append(*parts, sdk.RenderPart{Markdown: &body})
		renderparts.AppendText(parts, `</div>`)
	}
}

// parseTabTitle parses a top-level declaration such as === "Linux".
func parseTabTitle(line string) (string, bool) {
	if strings.TrimLeft(line, " \t") != line {
		return "", false
	}

	remaining, ok := strings.CutPrefix(strings.TrimSpace(line), "===")
	if !ok {
		return "", false
	}

	return pluginmarkdown.ParseQuotedTitle(strings.TrimSpace(remaining))
}
