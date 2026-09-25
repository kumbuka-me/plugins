package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
)

const (
	maxTaskDeclarations = 128
	maxTaskTokenBytes   = 2048
	maxTaskIDBytes      = 128
	maxTaskTextBytes    = 512
	maxAssigneeBytes    = 128
)

// taskOptions contains one parsed task declaration from page Markdown.
type taskOptions struct {
	// ID is the globally stable storage identity for the task.
	ID string
	// Text is the visible task description.
	Text string
	// Assignee optionally names the Kumbuka user responsible for the task.
	Assignee string
	// Due optionally contains a validated ISO calendar date.
	Due string
	// InitialState optionally selects a configured workflow state for a new task.
	InitialState string
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
func transformSource(source string, readStorage storageReader, workflow taskWorkflow) string {
	lines := strings.Split(source, "\n")
	var output strings.Builder
	fence := ""
	count := 0

	for index, line := range lines {
		if index > 0 {
			output.WriteByte('\n')
		}
		if fence != "" {
			output.WriteString(line)
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			continue
		}
		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			output.WriteString(line)
			continue
		}
		transformed, used := transformLine(line, maxTaskDeclarations-count, readStorage, workflow)
		count += used
		output.WriteString(transformed)
	}
	return output.String()
}

// transformLine renders task declarations on one non-fenced line while preserving inline code spans.
func transformLine(line string, remaining int, readStorage storageReader, workflow taskWorkflow) (string, int) {
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
		if codeTicks == 0 && used < remaining && strings.HasPrefix(line[index:], "{{task") {
			relativeEnd := strings.Index(line[index:], "}}")
			if relativeEnd < 0 {
				output.WriteString(taskErrorHTML("unterminated task declaration"))
				used++
				break
			}
			end := index + relativeEnd + 2
			token := line[index:end]
			if len(token) > maxTaskTokenBytes {
				output.WriteString(taskErrorHTML("task declaration is too long"))
			} else if options, err := parseTaskToken(token); err != nil {
				output.WriteString(taskErrorHTML(err.Error()))
			} else if _, err := workflow.initialState(options.InitialState); err != nil {
				output.WriteString(taskErrorHTML(err.Error()))
			} else {
				output.WriteString(renderTask(options, readStorage, workflow))
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
	arguments, ok := parseArguments(strings.TrimSpace(body))
	if !ok {
		return taskOptions{}, fmt.Errorf("invalid task attributes")
	}
	for name := range arguments {
		switch name {
		case "id", "text", "assignee", "due", "initial":
		default:
			return taskOptions{}, fmt.Errorf("unsupported task attribute %q", name)
		}
	}

	options := taskOptions{
		ID:           strings.TrimSpace(arguments["id"]),
		Text:         strings.TrimSpace(arguments["text"]),
		Assignee:     strings.TrimSpace(arguments["assignee"]),
		Due:          strings.TrimSpace(arguments["due"]),
		InitialState: strings.TrimSpace(arguments["initial"]),
	}
	if !validName(options.ID, maxTaskIDBytes) {
		return taskOptions{}, fmt.Errorf("task id must be 1-%d bytes and contain only letters, numbers, dots, underscores, hyphens, slashes, or colons", maxTaskIDBytes)
	}
	if options.Text == "" || len(options.Text) > maxTaskTextBytes || !utf8.ValidString(options.Text) {
		return taskOptions{}, fmt.Errorf("task text must be 1-%d bytes of valid UTF-8", maxTaskTextBytes)
	}
	if len(options.Assignee) > maxAssigneeBytes || !utf8.ValidString(options.Assignee) {
		return taskOptions{}, fmt.Errorf("task assignee must be at most %d bytes of valid UTF-8", maxAssigneeBytes)
	}
	if options.Assignee != "" && !validMention(options.Assignee) {
		return taskOptions{}, fmt.Errorf("task assignee must be a canonical @mention")
	}
	if options.Due != "" && !validDueDate(options.Due) {
		return taskOptions{}, fmt.Errorf("task due date must use YYYY-MM-DD")
	}
	if options.InitialState != "" && !validTaskStateID(options.InitialState) {
		return taskOptions{}, fmt.Errorf("task initial state must be a valid workflow state ID")
	}
	return options, nil
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
func renderTask(options taskOptions, read storageReader, workflow taskWorkflow) string {
	state := readTaskState(options, read, workflow)
	definition, ok := workflow.state(state.State)
	if !ok {
		return taskErrorHTML("task state is not configured")
	}

	completedClass := ""
	mark := ""
	if definition.Completed {
		completedClass = " kumbuka-task-completed"
		mark = "✓"
	}

	var output strings.Builder
	output.WriteString(`<span class="kumbuka-task-browser" data-kumbuka-plugin="me.kumbuka.tasks" data-kumbuka-module="task-ui" data-kumbuka-input="html">`)
	output.WriteString(`<span class="kumbuka-task-fallback` + completedClass + `" data-kumbuka-fallback>`)
	output.WriteString(`<span class="kumbuka-task-meta kumbuka-task-state__` + definition.ID + `"></span>`)
	output.WriteString(`<span class="kumbuka-task-choices" hidden>`)
	for _, choice := range workflow.States {
		output.WriteString(`<span class="kumbuka-task-choice kumbuka-task-choice-id__` + choice.ID)
		output.WriteString(` kumbuka-task-choice-action__` + actionID(options.ID, choice.ID))
		output.WriteString(` kumbuka-task-choice-color__` + strings.TrimPrefix(choice.Color, "#"))
		if choice.Completed {
			output.WriteString(` kumbuka-task-choice-completed__true`)
		} else {
			output.WriteString(` kumbuka-task-choice-completed__false`)
		}
		output.WriteString(`">` + html.EscapeString(choice.Label) + `</span>`)
	}
	output.WriteString(`</span>`)
	output.WriteString(`<span class="kumbuka-task-box" aria-hidden="true">` + mark + `</span>`)
	output.WriteString(`<span class="kumbuka-task-content"><span class="kumbuka-task-text">`)
	output.WriteString(html.EscapeString(options.Text))
	output.WriteString(`</span><span class="kumbuka-task-details">`)
	output.WriteString(`<span class="kumbuka-task-state-label">` + html.EscapeString(definition.Label) + `</span>`)
	if options.Assignee != "" {
		output.WriteString(`<span class="kumbuka-task-assignee">` + html.EscapeString(options.Assignee) + `</span>`)
	}
	if options.Due != "" {
		output.WriteString(`<span class="kumbuka-task-due">Due ` + html.EscapeString(options.Due) + `</span>`)
	}
	output.WriteString(`</span></span></span></span>`)
	return output.String()
}

// taskErrorHTML renders one escaped inline configuration error.
func taskErrorHTML(message string) string {
	escaped := html.EscapeString(message)
	return `<span class="kumbuka-task kumbuka-task-error" title="` + escaped + `">Task error: ` + escaped + `</span>`
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

// argumentParser incrementally parses a task macro argument list.
type argumentParser struct {
	// value is the complete argument string being parsed.
	value string
	// index is the next unread byte in value.
	index int
}

// parseArguments parses unique key=value options without regular expressions.
func parseArguments(value string) (map[string]string, bool) {
	parser := argumentParser{value: value}
	result := map[string]string{}
	for {
		name, value, done, ok := parser.next()
		if !ok {
			return nil, false
		}
		if done {
			return result, true
		}
		if _, exists := result[name]; exists {
			return nil, false
		}
		result[name] = value
	}
}

// next parses one name=value argument or reports the end of input.
func (p *argumentParser) next() (name, value string, done, ok bool) {
	p.skipSpace()
	if p.index == len(p.value) {
		return "", "", true, true
	}
	name = p.readName()
	if name == "" {
		return "", "", false, false
	}
	p.skipSpace()
	if p.index >= len(p.value) || p.value[p.index] != '=' {
		return "", "", false, false
	}
	p.index++
	p.skipSpace()
	value, ok = p.readValue()
	return name, value, false, ok
}

// skipSpace advances past spaces and horizontal tabs.
func (p *argumentParser) skipSpace() {
	for p.index < len(p.value) && (p.value[p.index] == ' ' || p.value[p.index] == '\t') {
		p.index++
	}
}

// readName consumes one lowercase option name.
func (p *argumentParser) readName() string {
	start := p.index
	for p.index < len(p.value) {
		character := p.value[p.index]
		if character >= 'a' && character <= 'z' || character == '_' {
			p.index++
			continue
		}
		break
	}
	return p.value[start:p.index]
}

// readValue consumes one quoted or unquoted option value.
func (p *argumentParser) readValue() (string, bool) {
	if p.index >= len(p.value) {
		return "", false
	}
	if p.value[p.index] == '"' {
		return p.readQuotedValue()
	}
	return p.readBareValue()
}

// readBareValue consumes a value up to the next horizontal whitespace.
func (p *argumentParser) readBareValue() (string, bool) {
	start := p.index
	for p.index < len(p.value) && p.value[p.index] != ' ' && p.value[p.index] != '\t' {
		p.index++
	}
	return p.value[start:p.index], p.index > start
}

// readQuotedValue consumes and unquotes a double-quoted option value.
func (p *argumentParser) readQuotedValue() (string, bool) {
	start := p.index
	p.index++
	escaped := false
	for p.index < len(p.value) {
		character := p.value[p.index]
		p.index++
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character == '"' {
			decoded, err := strconv.Unquote(p.value[start:p.index])
			return decoded, err == nil
		}
	}
	return "", false
}
