package main

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"net/url"
	"strconv"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const maxControlledStatuses = 8

// discoverStatuses returns unique status declarations from non-code Markdown source in document order.
func discoverStatuses(source string) []statusOptions {
	lines := strings.Split(source, "\n")
	seen := make(map[string]bool)
	result := make([]statusOptions, 0)
	var fence byte
	var fenceLength int

	for _, line := range lines {
		if len(result) >= maxControlledStatuses {
			break
		}
		if fence != 0 {
			if closesFence(line, fence, fenceLength) {
				fence = 0
				fenceLength = 0
			}
			continue
		}
		if marker, length := openingFence(line); marker != 0 {
			fence = marker
			fenceLength = length
			continue
		}

		for _, options := range statusTokens(line) {
			if seen[options.ID] {
				continue
			}
			seen[options.ID] = true
			result = append(result, options)
			if len(result) >= maxControlledStatuses {
				break
			}
		}
	}
	return result
}

// statusTokens parses status declarations on one line while ignoring inline code spans.
func statusTokens(line string) []statusOptions {
	var result []statusOptions
	codeTicks := 0
	for index := 0; index < len(line); {
		if line[index] == '`' {
			run := repeatedByte(line[index:], '`')
			if codeTicks == 0 {
				codeTicks = run
			} else if run == codeTicks {
				codeTicks = 0
			}
			index += run
			continue
		}
		if codeTicks == 0 && strings.HasPrefix(line[index:], "{{status") {
			end := strings.Index(line[index:], "}}")
			if end >= 0 {
				end += index + 2
				if options, err := parseStatusToken(line[index:end]); err == nil {
					result = append(result, options)
					index = end
					continue
				}
			}
		}
		index++
	}
	return result
}

// renderControls builds the page-details widget and host-rendered command actions.
func renderControls(source string, readResource resourceReader, readStorage storageReader) sdk.Result {
	options := discoverStatuses(source)
	if len(options) == 0 {
		return sdk.Text("")
	}

	var output strings.Builder
	output.WriteString(`<div class="status-controls"><h3>Status controls</h3><p class="muted">Change page statuses without editing the Markdown.</p>`)
	actions := make([]sdk.WidgetAction, 0)
	setCache := make(map[string]statusSet)

	for _, current := range options {
		set, err := resolveStatusSet(current, setCache, readResource)
		if err != nil {
			output.WriteString(`<div class="status-control status-control-error"><strong>`)
			output.WriteString(html.EscapeString(current.ID))
			output.WriteString(`</strong><small>`)
			output.WriteString(html.EscapeString(err.Error()))
			output.WriteString(`</small></div>`)
			continue
		}

		choice, err := selectedChoice(current, set, readStorage)
		if err != nil {
			output.WriteString(`<div class="status-control status-control-error"><strong>`)
			output.WriteString(html.EscapeString(current.ID))
			output.WriteString(`</strong><small>`)
			output.WriteString(html.EscapeString(err.Error()))
			output.WriteString(`</small></div>`)
			continue
		}
		label := current.Prefix
		if label == "" {
			label = current.ID
		}
		output.WriteString(`<div class="status-control"><strong>`)
		output.WriteString(html.EscapeString(label))
		output.WriteString(`</strong><small>Current: `)
		output.WriteString(html.EscapeString(choice.Label))
		output.WriteString(`</small></div>`)

		for index, candidate := range set.Choices {
			if candidate.Label == choice.Label {
				continue
			}
			actions = append(actions, sdk.WidgetAction{
				ID:    actionID(current.ID, index),
				Kind:  "command",
				Label: label + " → " + candidate.Label,
			})
		}
	}
	output.WriteString(`</div>`)

	result := sdk.Text(output.String())
	result.Actions = actions
	return result
}

// resolveAction maps a host command ID back to a currently declared and configured status choice.
func resolveAction(source, action string, readResource resourceReader) (string, string, bool) {
	setCache := make(map[string]statusSet)
	for _, options := range discoverStatuses(source) {
		set, err := resolveStatusSet(options, setCache, readResource)
		if err != nil {
			continue
		}
		for index, choice := range set.Choices {
			if action == actionID(options.ID, index) {
				return options.ID, choice.Label, true
			}
		}
	}
	return "", "", false
}

// actionID creates a host-valid opaque action identifier for one status choice.
func actionID(statusID string, choice int) string {
	sum := sha256.Sum256([]byte(statusID))
	return "set-" + hex.EncodeToString(sum[:12]) + "-" + strconv.Itoa(choice)
}

// pagePath escapes each canonical page slug segment for use in a local application URL.
func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
