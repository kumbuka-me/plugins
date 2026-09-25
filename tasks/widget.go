package main

import (
	"html"
	"net/url"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const maxControlledTasks = 32

// discoverTasks returns unique valid task declarations from non-code Markdown in document order.
func discoverTasks(source string) []taskOptions {
	lines := strings.Split(source, "\n")
	seen := make(map[string]bool)
	result := make([]taskOptions, 0)
	fence := ""

	for _, line := range lines {
		if len(result) >= maxControlledTasks {
			break
		}
		if fence != "" {
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			continue
		}
		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			continue
		}
		for _, options := range taskTokens(line) {
			if seen[options.ID] {
				continue
			}
			seen[options.ID] = true
			result = append(result, options)
			if len(result) >= maxControlledTasks {
				break
			}
		}
	}
	return result
}

// taskTokens parses task declarations on one line while ignoring inline code spans.
func taskTokens(line string) []taskOptions {
	var result []taskOptions
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
		if codeTicks == 0 && strings.HasPrefix(line[index:], "{{task") {
			end := strings.Index(line[index:], "}}")
			if end >= 0 {
				end += index + 2
				if options, err := parseTaskToken(line[index:end]); err == nil {
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

// renderControls builds the page-details task summary and host-rendered command actions.
func renderControls(source string, readStorage storageReader) sdk.Result {
	tasks := discoverTasks(source)
	if len(tasks) == 0 {
		return sdk.Text("")
	}

	var output strings.Builder
	output.WriteString(`<div class="task-controls"><h3>Tasks</h3><p class="muted">Track tasks on this page without editing its Markdown.</p>`)
	actions := make([]sdk.WidgetAction, 0, len(tasks))
	for _, task := range tasks {
		done := taskDone(task, readStorage)
		output.WriteString(`<div class="task-control"><strong>`)
		output.WriteString(html.EscapeString(task.Text))
		output.WriteString(`</strong><small>`)
		if done {
			output.WriteString(`Done`)
		} else {
			output.WriteString(`Open`)
		}
		if task.Assignee != "" {
			output.WriteString(` · ` + html.EscapeString(task.Assignee))
		}
		if task.Due != "" {
			output.WriteString(` · Due ` + html.EscapeString(task.Due))
		}
		output.WriteString(`</small></div>`)

		label := "Mark done"
		if done {
			label = "Reopen"
		}
		actions = append(actions, sdk.WidgetAction{
			ID:    actionID(task.ID, !done),
			Kind:  "command",
			Label: label + ": " + task.Text,
		})
	}
	output.WriteString(`</div>`)

	result := sdk.Text(output.String())
	result.Actions = actions
	return result
}

// resolveAction maps a host command ID back to a currently declared task and target state.
func resolveAction(source, action string) (string, bool, bool) {
	for _, task := range discoverTasks(source) {
		if action == actionID(task.ID, true) {
			return task.ID, true, true
		}
		if action == actionID(task.ID, false) {
			return task.ID, false, true
		}
	}
	return "", false, false
}

// pagePath escapes each canonical page slug segment for use in a local application URL.
func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
