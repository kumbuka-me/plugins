package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"strings"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const (
	maxControlledTasks            = 32
	maxTaskNotificationTitleBytes = 200
	maxTaskNotificationBodyBytes  = 2000
)

// taskMutationServices contains host operations used only during a task command.
type taskMutationServices struct {
	// Read loads the current plugin-owned task state.
	Read storageReader
	// Write persists a versioned task state.
	Write func(string, []byte) error
	// ResolveMention resolves a canonical assignee without exposing private user data.
	ResolveMention func(string) (sdk.User, error)
	// SendNotification creates an attributed core-owned notification.
	SendNotification func(sdk.NotificationInput) (sdk.Notification, error)
}

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
func resolveAction(source, action string) (taskOptions, bool, bool) {
	for _, task := range discoverTasks(source) {
		if action == actionID(task.ID, true) {
			return task, true, true
		}
		if action == actionID(task.ID, false) {
			return task, false, true
		}
	}
	return taskOptions{}, false, false
}

// applyTaskAction persists one transition and delivers its retry-safe assignee notification.
func applyTaskAction(page sdk.Page, source, action string, services taskMutationServices) error {
	task, done, ok := resolveAction(source, action)
	if !ok {
		return fmt.Errorf("task action is no longer available")
	}
	if services.Read == nil || services.Write == nil {
		return fmt.Errorf("task storage is unavailable")
	}

	state, err := loadTaskState(task, services.Read)
	if err != nil {
		return err
	}
	if state.Done != done {
		if state.Version == ^uint64(0) {
			return fmt.Errorf("task transition limit reached")
		}
		state.Done = done
		state.Version++
		state.NotificationSent = task.Assignee == ""
		if err := writeTaskState(task.ID, state, services.Write); err != nil {
			return err
		}
	}
	if task.Assignee == "" {
		if !state.NotificationSent {
			state.NotificationSent = true
			return writeTaskState(task.ID, state, services.Write)
		}
		return nil
	}
	if state.NotificationSent {
		return nil
	}
	if services.ResolveMention == nil || services.SendNotification == nil {
		return fmt.Errorf("task notification capability is unavailable")
	}

	assignee, err := services.ResolveMention(task.Assignee)
	if err != nil {
		return err
	}
	title := taskNotificationTitle(done, task.Text)
	body := "A task on " + page.Title + " was reopened."
	if done {
		body = "A task on " + page.Title + " was marked complete."
	}
	body = boundedUTF8(body, maxTaskNotificationBodyBytes)
	_, err = services.SendNotification(sdk.NotificationInput{
		RecipientUserID: assignee.ID,
		Title:           title,
		Body:            body,
		URL:             "/pages/" + pagePath(page.Slug),
		IdempotencyKey:  taskNotificationKey(page.Slug, task.ID, state),
	})
	if err != nil {
		return err
	}
	state.NotificationSent = true
	return writeTaskState(task.ID, state, services.Write)
}

// loadTaskState reads task state for a mutation without hiding persistence failures.
func loadTaskState(task taskOptions, read storageReader) (taskState, error) {
	stored, err := read(storageKey(task.ID))
	if err != nil {
		return taskState{}, err
	}
	return decodeTaskState(task, stored), nil
}

// taskNotificationTitle creates a bounded UTF-8 notification heading.
func taskNotificationTitle(done bool, text string) string {
	prefix := "Task reopened: "
	if done {
		prefix = "Task completed: "
	}
	return boundedUTF8(prefix+text, maxTaskNotificationTitleBytes)
}

// boundedUTF8 truncates text to a byte limit without splitting a UTF-8 sequence.
func boundedUTF8(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return strings.TrimSpace(value[:limit])
}

// writeTaskState encodes and persists one bounded task state value.
func writeTaskState(taskID string, state taskState, write func(string, []byte) error) error {
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return write(storageKey(taskID), value)
}

// taskNotificationKey returns a stable idempotency key for one page task transition.
func taskNotificationKey(pageSlug, taskID string, state taskState) string {
	value := fmt.Sprintf("%s\x00%s\x00%d\x00%t", pageSlug, taskID, state.Version, state.Done)
	sum := sha256.Sum256([]byte(value))
	return "task-transition-" + hex.EncodeToString(sum[:16])
}

// pagePath escapes each canonical page slug segment for use in a local application URL.
func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
