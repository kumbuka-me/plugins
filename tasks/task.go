package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kumbuka-me/plugins/internal/macroargs"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const (
	maxTaskDeclarations     = 128
	maxTaskTokenBytes       = 8192
	maxTaskListTokenBytes   = 192 * 1024
	maxTaskIDBytes          = 128
	maxTaskTextBytes        = 512
	maxTaskDescriptionBytes = 4096
	maxAssigneeBytes        = 128
	maxTaskDepth            = 16
	taskListSeparator       = "\x1f"
)

// taskOptions contains one parsed task declaration from page Markdown.
type taskOptions struct {
	// ID is the globally stable storage identity for the task.
	ID string
	// Text is the visible task title.
	Text string
	// Description optionally contains longer task context.
	Description string
	// Assignee optionally names the Kumbuka user responsible for the task.
	Assignee string
	// Due optionally contains a validated ISO calendar date.
	Due string
	// InitialState optionally selects a configured workflow state for a new task.
	InitialState string
	// Parent optionally names an earlier task in the same rendered list.
	Parent string
}

// storageReader reads one plugin-owned persisted task value.
type storageReader func(key string) (sdk.StoredValue, error)

// taskState contains one persisted task state transition.
type taskState struct {
	// State is the configured workflow state ID currently assigned to the task.
	State string `json:"state,omitempty"`
	// Version increments for each state transition and scopes notification idempotency.
	Version uint64 `json:"version"`
	// NotificationSent reports whether the current transition notification completed.
	NotificationSent bool `json:"notification_sent"`
	// PreviousState preserves the transition origin until its notification is durably acknowledged.
	PreviousState string `json:"previous_state,omitempty"`
	// LegacyDone decodes the pre-workflow boolean state representation.
	LegacyDone *bool `json:"done,omitempty"`
}

// transformSource renders task declarations outside fenced and inline code.
func transformSource(source string, readStorage storageReader, workflow taskWorkflow, localizer sdk.Localizer) string {
	lines := strings.Split(source, "\n")
	output := make([]string, 0, len(lines))
	fence := ""
	count := 0

	for index := 0; index < len(lines); {
		line := lines[index]
		if fence != "" {
			output = append(output, line)
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			index++
			continue
		}
		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			output = append(output, line)
			index++
			continue
		}

		if tasks, next, ok := collectTaskList(lines, index, maxTaskDeclarations-count, workflow); ok {
			output = append(output, renderTaskList(tasks, readStorage, workflow, localizer))
			count += len(tasks)
			index = next
			continue
		}

		transformed, used := transformLine(line, maxTaskDeclarations-count, readStorage, workflow, localizer)
		count += used
		output = append(output, transformed)
		index++
	}

	return strings.Join(output, "\n")
}

// collectTaskList groups adjacent standalone task declarations into one compact rendered list.
// Blank lines between task declarations are treated as list spacing and consumed.
func collectTaskList(lines []string, start, remaining int, workflow taskWorkflow) ([]taskOptions, int, bool) {
	if remaining <= 0 || start >= len(lines) {
		return nil, start, false
	}

	first, ok := standaloneTask(lines[start], workflow)
	if !ok {
		return nil, start, false
	}

	tasks := []taskOptions{first}
	next := start + 1
	for len(tasks) < remaining {
		separatorStart := next
		for next < len(lines) && strings.TrimSpace(lines[next]) == "" {
			next++
		}
		if next >= len(lines) {
			next = separatorStart
			break
		}

		task, found := standaloneTask(lines[next], workflow)
		if !found {
			next = separatorStart
			break
		}
		tasks = append(tasks, task)
		next++
	}

	return tasks, next, true
}

// standaloneTask parses a line that consists only of one valid task declaration.
func standaloneTask(line string, workflow taskWorkflow) (taskOptions, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "{{task") || !strings.HasSuffix(trimmed, "}}") {
		return taskOptions{}, false
	}
	options, err := parseTaskToken(trimmed)
	if err != nil {
		return taskOptions{}, false
	}
	if _, err := workflow.initialState(options.InitialState); err != nil {
		return taskOptions{}, false
	}
	return options, true
}

// transformLine renders task declarations on one non-fenced line while preserving inline code spans.
func transformLine(line string, remaining int, readStorage storageReader, workflow taskWorkflow, localizer sdk.Localizer) (string, int) {
	if remaining <= 0 || !strings.Contains(line, "{{task") {
		return line, 0
	}

	var output strings.Builder
	used := 0
	codeTicks := 0
	for index := 0; index < len(line); {
		if line[index] == '`' {
			run := repeatedByte(line[index:], '`')
			output.WriteString(line[index : index+run])
			if codeTicks == 0 {
				codeTicks = run
			} else if run == codeTicks {
				codeTicks = 0
			}
			index += run
			continue
		}

		if codeTicks == 0 && used < remaining && strings.HasPrefix(line[index:], "{{tasks") {
			relativeEnd := strings.Index(line[index:], "}}")
			if relativeEnd < 0 {
				output.WriteString(taskErrorHTML("unterminated task list declaration", localizer))
				used++
				break
			}
			end := index + relativeEnd + 2
			token := line[index:end]
			if len(token) > maxTaskListTokenBytes {
				output.WriteString(taskErrorHTML("task list declaration is too long", localizer))
				used++
			} else if tasks, err := parseTaskListToken(token); err != nil {
				output.WriteString(taskErrorHTML(err.Error(), localizer))
				used++
			} else if len(tasks) > remaining-used {
				output.WriteString(taskErrorHTML(fmt.Sprintf("task list exceeds the %d task page limit", maxTaskDeclarations), localizer))
				used = remaining
			} else if err := validateTaskWorkflowStates(tasks, workflow); err != nil {
				output.WriteString(taskErrorHTML(err.Error(), localizer))
				used += len(tasks)
			} else {
				output.WriteString(renderTaskList(tasks, readStorage, workflow, localizer))
				used += len(tasks)
			}
			index = end
			continue
		}

		if codeTicks == 0 && used < remaining && strings.HasPrefix(line[index:], "{{task") {
			relativeEnd := strings.Index(line[index:], "}}")
			if relativeEnd < 0 {
				output.WriteString(taskErrorHTML("unterminated task declaration", localizer))
				used++
				break
			}
			end := index + relativeEnd + 2
			token := line[index:end]
			if len(token) > maxTaskTokenBytes {
				output.WriteString(taskErrorHTML("task declaration is too long", localizer))
			} else if options, err := parseTaskToken(token); err != nil {
				output.WriteString(taskErrorHTML(err.Error(), localizer))
			} else if _, err := workflow.initialState(options.InitialState); err != nil {
				output.WriteString(taskErrorHTML(err.Error(), localizer))
			} else {
				output.WriteString(renderTask(options, readStorage, workflow, localizer))
			}
			used++
			index = end
			continue
		}

		output.WriteByte(line[index])
		index++
	}
	return output.String(), used
}

// parseTaskListToken parses one visual-editor task list declaration into individual task options.
func parseTaskListToken(token string) ([]taskOptions, error) {
	value := strings.TrimSpace(token)
	body, ok := strings.CutPrefix(value, "{{tasks")
	if !ok || len(value) > maxTaskListTokenBytes {
		return nil, fmt.Errorf("invalid task list declaration")
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok || (body != "" && body[0] != ' ' && body[0] != '\t') {
		return nil, fmt.Errorf("invalid task list declaration")
	}

	arguments, ok := macroargs.ParseUnique(strings.TrimSpace(body))
	if !ok {
		return nil, fmt.Errorf("invalid task list attributes")
	}
	for name := range arguments {
		switch name {
		case "texts", "descriptions", "ids", "parents", "assignees", "dues", "initials":
		default:
			return nil, fmt.Errorf("unsupported task list attribute %q", name)
		}
	}

	texts := splitTaskListAttribute(arguments["texts"])
	if len(texts) == 0 {
		return nil, fmt.Errorf("task list must contain at least one task text")
	}
	ids, err := optionalTaskListAttribute(arguments["ids"], len(texts), "ids")
	if err != nil {
		return nil, err
	}
	for index := range ids {
		if strings.TrimSpace(ids[index]) == "" {
			ids[index] = automaticTaskID(index, texts[index])
		}
	}
	if len(texts) > maxTaskDeclarations {
		return nil, fmt.Errorf("task list may contain at most %d tasks", maxTaskDeclarations)
	}

	descriptions, err := optionalTaskListAttribute(arguments["descriptions"], len(texts), "descriptions")
	if err != nil {
		return nil, err
	}
	parents, err := optionalTaskListAttribute(arguments["parents"], len(texts), "parents")
	if err != nil {
		return nil, err
	}
	assignees, err := optionalTaskListAttribute(arguments["assignees"], len(texts), "assignees")
	if err != nil {
		return nil, err
	}
	dues, err := optionalTaskListAttribute(arguments["dues"], len(texts), "dues")
	if err != nil {
		return nil, err
	}
	initials, err := optionalTaskListAttribute(arguments["initials"], len(texts), "initials")
	if err != nil {
		return nil, err
	}

	tasks := make([]taskOptions, len(texts))
	for index := range texts {
		options := taskOptions{
			ID:           strings.TrimSpace(ids[index]),
			Text:         strings.TrimSpace(texts[index]),
			Description:  strings.TrimSpace(descriptions[index]),
			Parent:       strings.TrimSpace(parents[index]),
			Assignee:     strings.TrimSpace(assignees[index]),
			Due:          strings.TrimSpace(dues[index]),
			InitialState: strings.TrimSpace(initials[index]),
		}
		if err := validateTaskOptions(options); err != nil {
			return nil, fmt.Errorf("task %d: %w", index+1, err)
		}
		tasks[index] = options
	}
	if _, err := taskDepths(tasks); err != nil {
		return nil, err
	}
	return tasks, nil
}

// splitTaskListAttribute decodes one visual-editor list attribute while preserving empty cells.
func splitTaskListAttribute(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, taskListSeparator)
}

// optionalTaskListAttribute pads an omitted or short optional list with blank values.
func optionalTaskListAttribute(value string, count int, name string) ([]string, error) {
	values := splitTaskListAttribute(value)
	if len(values) > count {
		return nil, fmt.Errorf("task list %s contains more values than tasks", name)
	}
	padded := make([]string, count)
	copy(padded, values)
	return padded, nil
}

// validateTaskWorkflowStates verifies every requested initial state against the active workflow.
func validateTaskWorkflowStates(tasks []taskOptions, workflow taskWorkflow) error {
	for _, task := range tasks {
		if _, err := workflow.initialState(task.InitialState); err != nil {
			return err
		}
	}
	return nil
}

// parseTaskToken parses one complete task declaration and validates its attributes.
func parseTaskToken(token string) (taskOptions, error) {
	value := strings.TrimSpace(token)
	body, ok := strings.CutPrefix(value, "{{task")
	if !ok || len(value) > maxTaskTokenBytes {
		return taskOptions{}, fmt.Errorf("invalid task declaration")
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok || (body != "" && body[0] != ' ' && body[0] != '\t') {
		return taskOptions{}, fmt.Errorf("invalid task declaration")
	}
	arguments, ok := macroargs.ParseUnique(strings.TrimSpace(body))
	if !ok {
		return taskOptions{}, fmt.Errorf("invalid task attributes")
	}
	for name := range arguments {
		switch name {
		case "id", "text", "description", "assignee", "due", "initial", "parent":
		default:
			return taskOptions{}, fmt.Errorf("unsupported task attribute %q", name)
		}
	}

	options := taskOptions{
		ID:           strings.TrimSpace(arguments["id"]),
		Text:         strings.TrimSpace(arguments["text"]),
		Description:  strings.TrimSpace(arguments["description"]),
		Assignee:     strings.TrimSpace(arguments["assignee"]),
		Due:          strings.TrimSpace(arguments["due"]),
		InitialState: strings.TrimSpace(arguments["initial"]),
		Parent:       strings.TrimSpace(arguments["parent"]),
	}
	if err := validateTaskOptions(options); err != nil {
		return taskOptions{}, err
	}
	return options, nil
}

// validateTaskOptions validates the shared fields used by legacy task and task-list declarations.
func validateTaskOptions(options taskOptions) error {
	if !validName(options.ID, maxTaskIDBytes) {
		return fmt.Errorf("task id must be 1-%d bytes and contain only letters, numbers, dots, underscores, hyphens, slashes, or colons", maxTaskIDBytes)
	}
	if options.Text == "" || len(options.Text) > maxTaskTextBytes || !utf8.ValidString(options.Text) {
		return fmt.Errorf("task text must be 1-%d bytes of valid UTF-8", maxTaskTextBytes)
	}
	if len(options.Description) > maxTaskDescriptionBytes || !utf8.ValidString(options.Description) {
		return fmt.Errorf("task description must be at most %d bytes of valid UTF-8", maxTaskDescriptionBytes)
	}
	if len(options.Assignee) > maxAssigneeBytes || !utf8.ValidString(options.Assignee) {
		return fmt.Errorf("task assignee must be at most %d bytes of valid UTF-8", maxAssigneeBytes)
	}
	if options.Assignee != "" && !validMention(options.Assignee) {
		return fmt.Errorf("task assignee must be a canonical @mention")
	}
	if options.Due != "" && !validDueDate(options.Due) {
		return fmt.Errorf("task due date must use YYYY-MM-DD")
	}
	if options.InitialState != "" && !validTaskStateID(options.InitialState) {
		return fmt.Errorf("task initial state must be a valid workflow state ID")
	}
	if options.Parent != "" && !validName(options.Parent, maxTaskIDBytes) {
		return fmt.Errorf("task parent must be a valid task id")
	}
	if options.Parent == options.ID {
		return fmt.Errorf("task cannot be its own parent")
	}
	return nil
}

// automaticTaskID derives a stable fallback identity for a task-list row that has not been saved by the visual editor yet.
func automaticTaskID(index int, text string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d\x00%s", index, text)))
	return "task-" + hex.EncodeToString(sum[:12])
}

// validMention reports whether value is a bounded canonical Kumbuka mention.
func validMention(value string) bool {
	if len(value) < 2 || len(value) > maxAssigneeBytes || value[0] != '@' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !mentionNameByte(value[index]) {
			return false
		}
	}
	return true
}

// mentionNameByte reports whether character is allowed in a Kumbuka username.
func mentionNameByte(character byte) bool {
	return asciiLetterOrDigit(character) || character == '-' || character == '_' || character == '.'
}

// validDueDate reports whether value is a canonical YYYY-MM-DD calendar date.
func validDueDate(value string) bool {
	if len(value) != 10 {
		return false
	}
	parsed, err := time.Parse("2006-01-02", value)
	return err == nil && parsed.Format("2006-01-02") == value
}

// validName reports whether value is a bounded task identifier.
func validName(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		if !taskNameByte(value[index]) {
			return false
		}
	}
	return true
}

// taskNameByte reports whether character is allowed in a stable task identifier.
func taskNameByte(character byte) bool {
	return asciiLetterOrDigit(character) || character == '-' || character == '_' || character == '.' || character == '/' || character == ':'
}

// asciiLetterOrDigit reports whether character is an ASCII letter or decimal digit.
func asciiLetterOrDigit(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z' || character >= '0' && character <= '9'
}

// readTaskState resolves persisted state while accepting the original open/done storage formats.
func readTaskState(options taskOptions, read storageReader, workflow taskWorkflow) taskState {
	initial, err := workflow.initialState(options.InitialState)
	if err != nil {
		initial = workflow.legacyState(false)
	}
	fallback := taskState{State: initial.ID, NotificationSent: true}
	if read == nil {
		return fallback
	}
	stored, err := read(storageKey(options.ID))
	if err != nil {
		return fallback
	}
	return decodeTaskState(options, stored, workflow)
}

// decodeTaskState decodes current and legacy persisted task state values.
func decodeTaskState(options taskOptions, stored sdk.StoredValue, workflow taskWorkflow) taskState {
	initial, err := workflow.initialState(options.InitialState)
	if err != nil {
		initial = workflow.legacyState(false)
	}
	fallback := taskState{State: initial.ID, NotificationSent: true}
	if !stored.Found {
		return fallback
	}
	if string(stored.Value) == "done" {
		legacy := workflow.legacyState(true)
		return taskState{State: legacy.ID, NotificationSent: true}
	}
	if string(stored.Value) == "open" {
		legacy := workflow.legacyState(false)
		return taskState{State: legacy.ID, NotificationSent: true}
	}

	var state taskState
	if json.Unmarshal(stored.Value, &state) != nil || state.Version == 0 {
		return fallback
	}
	if state.State == "" && state.LegacyDone != nil {
		state.State = workflow.legacyState(*state.LegacyDone).ID
	}
	if _, ok := workflow.state(state.State); !ok {
		return fallback
	}
	state.LegacyDone = nil
	return state
}

// renderTask renders one task declaration into safe fallback HTML and browser-module metadata.
func renderTask(options taskOptions, read storageReader, workflow taskWorkflow, localizer sdk.Localizer) string {
	return renderTaskList([]taskOptions{options}, read, workflow, localizer)
}

// renderTaskList renders one or more tasks as a compact nested list.
func renderTaskList(tasks []taskOptions, read storageReader, workflow taskWorkflow, localizer sdk.Localizer) string {
	if len(tasks) == 0 {
		return ""
	}

	depths, err := taskDepths(tasks)
	if err != nil {
		return taskErrorHTML(err.Error(), localizer)
	}

	states := make([]taskWorkflowState, len(tasks))
	completed := 0
	for index, options := range tasks {
		state := readTaskState(options, read, workflow)
		definition, ok := workflow.state(state.State)
		if !ok {
			return taskErrorHTML("task state is not configured", localizer)
		}
		states[index] = definition
		if definition.Completed {
			completed++
		}
	}

	roots, children := taskTree(tasks)
	listClass := "kumbuka-task-list"
	if len(tasks) == 1 {
		listClass += " kumbuka-task-list-single"
	}

	var output strings.Builder
	output.WriteString(`<span class="kumbuka-task-browser" data-kumbuka-plugin="me.kumbuka.tasks" data-kumbuka-module="task-ui" data-kumbuka-input="html">`)
	output.WriteString(`<span class="` + listClass + `" data-kumbuka-fallback>`)
	if len(tasks) > 1 {
		output.WriteString(renderTaskListHeader(localizer.Text("tasks.title"), completed, len(tasks)))
	}
	output.WriteString(`<span class="kumbuka-task-items">`)
	for _, index := range roots {
		renderTaskNode(&output, index, tasks, states, depths, children, workflow, localizer)
	}
	output.WriteString(`</span></span></span>`)
	return output.String()
}

// renderTaskListHeader renders the list title and progress counter as one complete fragment.
// Keeping the counter in one fragment prevents an incomplete header when the renderer changes.
func renderTaskListHeader(title string, completed, total int) string {
	return fmt.Sprintf(
		`<span class="kumbuka-task-list-header"><span class="kumbuka-task-list-title">%s</span><span class="kumbuka-task-progress"><span class="kumbuka-task-progress-current">%d</span><span class="kumbuka-task-progress-separator"> / </span><span class="kumbuka-task-progress-total">%d</span></span></span>`,
		html.EscapeString(title),
		completed,
		total,
	)
}

// taskDepths resolves parent references in document order and rejects invalid task trees.
func taskDepths(tasks []taskOptions) ([]int, error) {
	depths := make([]int, len(tasks))
	byID := make(map[string]int, len(tasks))

	for index, task := range tasks {
		if _, exists := byID[task.ID]; exists {
			return nil, fmt.Errorf("task id %q is duplicated in the same task list", task.ID)
		}

		depth := 0
		if task.Parent != "" {
			parentDepth, found := byID[task.Parent]
			if !found {
				return nil, fmt.Errorf("task parent %q must appear before its subtask in the same task list", task.Parent)
			}
			depth = parentDepth + 1
			if depth > maxTaskDepth {
				return nil, fmt.Errorf("task nesting may not exceed %d levels", maxTaskDepth)
			}
		}

		depths[index] = depth
		byID[task.ID] = depth
	}
	return depths, nil
}

// taskTree converts validated parent references into root and child indexes while preserving row order.
func taskTree(tasks []taskOptions) ([]int, [][]int) {
	roots := make([]int, 0, len(tasks))
	children := make([][]int, len(tasks))
	byID := make(map[string]int, len(tasks))
	for index, task := range tasks {
		if task.Parent == "" {
			roots = append(roots, index)
		} else if parent, ok := byID[task.Parent]; ok {
			children[parent] = append(children[parent], index)
		}
		byID[task.ID] = index
	}
	return roots, children
}

// renderTaskNode writes one task row followed by recursively nested child tasks.
func renderTaskNode(output *strings.Builder, index int, tasks []taskOptions, states []taskWorkflowState, depths []int, children [][]int, workflow taskWorkflow, localizer sdk.Localizer) {
	output.WriteString(`<span class="kumbuka-task-node kumbuka-task-depth__` + fmt.Sprintf("%d", depths[index]) + `">`)
	output.WriteString(renderTaskItem(tasks[index], states[index], workflow, localizer))
	if len(children[index]) > 0 {
		output.WriteString(`<span class="kumbuka-task-children">`)
		for _, child := range children[index] {
			renderTaskNode(output, child, tasks, states, depths, children, workflow, localizer)
		}
		output.WriteString(`</span>`)
	}
	output.WriteString(`</span>`)
}

// renderTaskItem renders one passive task row and sanitizer-safe browser command metadata.
func renderTaskItem(options taskOptions, definition taskWorkflowState, workflow taskWorkflow, localizer sdk.Localizer) string {
	mark := ""
	if definition.Completed {
		mark = "✓"
	}

	classes := []string{"kumbuka-task-fallback", "kumbuka-task-state__" + definition.ID}
	if definition.Completed {
		classes = append(classes, "kumbuka-task-completed")
	}
	for _, choice := range workflow.States {
		classes = append(classes, taskChoiceClass(options.ID, choice, workflow.stateLabel(choice, localizer)))
	}

	var output strings.Builder
	output.WriteString(`<span class="` + strings.Join(classes, " ") + `">`)
	output.WriteString(`<span class="kumbuka-task-box" aria-hidden="true">` + mark + `</span>`)
	output.WriteString(`<span class="kumbuka-task-content"><span class="kumbuka-task-text">`)
	output.WriteString(html.EscapeString(options.Text))
	output.WriteString(`</span>`)
	if options.Description != "" {
		output.WriteString(`<span class="kumbuka-task-description">` + html.EscapeString(options.Description) + `</span>`)
	}
	output.WriteString(`<span class="kumbuka-task-details">`)
	output.WriteString(`<span class="kumbuka-task-state-label">` + html.EscapeString(workflow.stateLabel(definition, localizer)) + `</span>`)
	if options.Assignee != "" {
		output.WriteString(`<span class="kumbuka-task-assignee" data-kumbuka-mention>` + html.EscapeString(options.Assignee) + `</span>`)
	}
	if options.Due != "" {
		output.WriteString(`<span class="kumbuka-task-due">` + html.EscapeString(localizer.Textf("tasks.due", options.Due)) + `</span>`)
	}
	output.WriteString(`</span></span></span>`)
	return output.String()
}

// taskChoiceClass encodes workflow metadata into one sanitizer-safe class token.
func taskChoiceClass(taskID string, choice taskWorkflowState, label string) string {
	completed := "0"
	if choice.Completed {
		completed = "1"
	}
	return "kumbuka-task-choice__" +
		hex.EncodeToString([]byte(choice.ID)) + "__" +
		actionID(taskID, choice.ID) + "__" +
		strings.TrimPrefix(choice.Color, "#") + "__" +
		completed + "__" +
		hex.EncodeToString([]byte(label))
}

// taskErrorHTML renders one escaped inline configuration error.
func taskErrorHTML(message string, localizer sdk.Localizer) string {
	escaped := html.EscapeString(message)
	return `<span class="kumbuka-task kumbuka-task-error" title="` + escaped + `">` + html.EscapeString(localizer.Textf("tasks.error", message)) + `</span>`
}

// storageKey derives a bounded opaque storage key from a public task identifier.
func storageKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "task." + hex.EncodeToString(sum[:16])
}

// actionID creates a host-valid opaque action identifier for one task state transition.
func actionID(taskID, stateID string) string {
	sum := sha256.Sum256([]byte(taskID + "\x00" + stateID))
	return "task-" + hex.EncodeToString(sum[:12])
}

// repeatedByte returns the length of the leading run of the requested byte.
func repeatedByte(value string, target byte) int {
	count := 0
	for count < len(value) && value[count] == target {
		count++
	}
	return count
}
