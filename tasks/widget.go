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

// taskAssignmentServices contains host operations available to committed page-content hooks.
type taskAssignmentServices struct {
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

// renderControls builds the page-details task summary and one non-JavaScript advance action per task.
func renderControls(source string, readStorage storageReader, workflow taskWorkflow) sdk.Result {
	tasks := discoverTasks(source)
	if len(tasks) == 0 {
		return sdk.Text("")
	}

	var output strings.Builder
	output.WriteString(`<div class="task-controls"><h3>Tasks</h3><p class="muted">Track tasks on this page without editing its Markdown.</p>`)
	actions := make([]sdk.WidgetAction, 0, len(tasks))
	for _, task := range tasks {
		state := readTaskState(task, readStorage, workflow)
		definition, ok := workflow.state(state.State)
		if !ok {
			continue
		}
		output.WriteString(`<div class="task-control"><strong>`)
		output.WriteString(html.EscapeString(task.Text))
		output.WriteString(`</strong><small>`)
		output.WriteString(html.EscapeString(definition.Label))
		if task.Assignee != "" {
			output.WriteString(` · ` + html.EscapeString(task.Assignee))
		}
		if task.Due != "" {
			output.WriteString(` · Due ` + html.EscapeString(task.Due))
		}
		output.WriteString(`</small></div>`)

		if len(workflow.States) > 1 {
			next := workflow.nextState(definition.ID)
			actions = append(actions, sdk.WidgetAction{
				ID:    actionID(task.ID, next.ID),
				Kind:  "command",
				Label: "Move to " + next.Label + ": " + task.Text,
			})
		}
	}
	output.WriteString(`</div>`)

	result := sdk.Text(output.String())
	result.Actions = actions
	return result
}

// resolveAction maps a host command ID back to a currently declared task and configured target state.
func resolveAction(source, action string, workflow taskWorkflow) (taskOptions, taskWorkflowState, bool) {
	for _, task := range discoverTasks(source) {
		for _, state := range workflow.States {
			if action == actionID(task.ID, state.ID) {
				return task, state, true
			}
		}
	}
	return taskOptions{}, taskWorkflowState{}, false
}

// applyTaskAction persists one transition and delivers its retry-safe assignee notification.
func applyTaskAction(page sdk.Page, source, action string, workflow taskWorkflow, services taskMutationServices) error {
	task, target, ok := resolveAction(source, action, workflow)
	if !ok {
		return fmt.Errorf("task action is no longer available")
	}
	if services.Read == nil || services.Write == nil {
		return fmt.Errorf("task storage is unavailable")
	}

	state, err := loadTaskState(task, services.Read, workflow)
	if err != nil {
		return err
	}
	previous, ok := workflow.state(state.State)
	if !ok {
		return fmt.Errorf("task state is no longer configured")
	}
	if state.State != target.ID {
		if state.Version == ^uint64(0) {
			return fmt.Errorf("task transition limit reached")
		}
		state.PreviousState = state.State
		state.State = target.ID
		state.Version++
		state.NotificationSent = task.Assignee == ""
		if state.NotificationSent {
			state.PreviousState = ""
		}
		if err := writeTaskState(task.ID, state, services.Write); err != nil {
			return err
		}
	} else if state.PreviousState != "" {
		if transitionOrigin, found := workflow.state(state.PreviousState); found {
			previous = transitionOrigin
		}
	}
	if task.Assignee == "" {
		if !state.NotificationSent {
			state.NotificationSent = true
			state.PreviousState = ""
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
	title, body := taskStateNotification(previous, target, task.Text, page.Title)
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
	state.PreviousState = ""
	return writeTaskState(task.ID, state, services.Write)
}

// notifyTaskAssignments sends one retry-safe notification for each new or changed assignee.
func notifyTaskAssignments(context sdk.ContentChangeContext, services taskAssignmentServices) error {
	if services.ResolveMention == nil || services.SendNotification == nil {
		return fmt.Errorf("task assignment notification capability is unavailable")
	}

	previous := make(map[string]string)
	for _, task := range discoverTasks(context.PreviousSource) {
		previous[task.ID] = task.Assignee
	}
	for _, task := range discoverTasks(context.Source) {
		if task.Assignee == "" || previous[task.ID] == task.Assignee {
			continue
		}
		assignee, err := services.ResolveMention(task.Assignee)
		if err != nil {
			return err
		}
		_, err = services.SendNotification(sdk.NotificationInput{
			RecipientUserID: assignee.ID,
			Title:           boundedUTF8("Task assigned: "+task.Text, maxTaskNotificationTitleBytes),
			Body:            boundedUTF8("You were assigned a task on "+context.Page.Title+".", maxTaskNotificationBodyBytes),
			URL:             "/pages/" + pagePath(context.Page.Slug),
			IdempotencyKey:  taskAssignmentNotificationKey(context.Page, task),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// loadTaskState reads task state for a mutation without hiding persistence failures.
func loadTaskState(task taskOptions, read storageReader, workflow taskWorkflow) (taskState, error) {
	stored, err := read(storageKey(task.ID))
	if err != nil {
		return taskState{}, err
	}
	return decodeTaskState(task, stored, workflow), nil
}

// taskStateNotification creates a bounded notification for one configured state transition.
func taskStateNotification(previous, target taskWorkflowState, text, pageTitle string) (string, string) {
	prefix := "Task state changed: "
	body := "A task on " + pageTitle + " moved to " + target.Label + "."
	switch {
	case target.Completed && !previous.Completed:
		prefix = "Task completed: "
		body = "A task on " + pageTitle + " was completed as " + target.Label + "."
	case previous.Completed && !target.Completed:
		prefix = "Task reopened: "
		body = "A task on " + pageTitle + " was reopened as " + target.Label + "."
	}
	return boundedUTF8(prefix+text, maxTaskNotificationTitleBytes), boundedUTF8(body, maxTaskNotificationBodyBytes)
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
	state.LegacyDone = nil
	value, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return write(storageKey(taskID), value)
}

// taskNotificationKey returns a stable idempotency key for one page task transition.
func taskNotificationKey(pageSlug, taskID string, state taskState) string {
	value := fmt.Sprintf("%s\x00%s\x00%d\x00%s", pageSlug, taskID, state.Version, state.State)
	sum := sha256.Sum256([]byte(value))
	return "task-transition-" + hex.EncodeToString(sum[:16])
}

// taskAssignmentNotificationKey returns a stable key for one committed assignment version.
func taskAssignmentNotificationKey(page sdk.Page, task taskOptions) string {
	value := fmt.Sprintf("%s\x00%s\x00%s\x00%d", page.Slug, task.ID, task.Assignee, page.UpdatedAt.UnixNano())
	sum := sha256.Sum256([]byte(value))
	return "task-assignment-" + hex.EncodeToString(sum[:16])
}

// pagePath escapes each canonical page slug segment for use in a local application URL.
func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
