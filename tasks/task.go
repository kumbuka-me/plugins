package main

import (
	"crypto/sha256"
	"encoding/hex"
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
	// Assignee optionally names the person, team, or role responsible.
	Assignee string
	// Due optionally contains a validated ISO calendar date.
	Due string
	// InitialDone is the fallback completion state before persisted state exists.
	InitialDone bool
}

// storageReader reads one plugin-owned persisted task value.
type storageReader func(key string) (sdk.StoredValue, error)

// transformSource renders task declarations outside fenced and inline code.
func transformSource(source string, readStorage storageReader) string {
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
		transformed, used := transformLine(line, maxTaskDeclarations-count, readStorage)
		count += used
		output.WriteString(transformed)
	}
	return output.String()
}

// transformLine renders task declarations on one non-fenced line while preserving inline code spans.
func transformLine(line string, remaining int, readStorage storageReader) (string, int) {
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
			} else {
				output.WriteString(renderTask(options, readStorage))
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
		ID:       strings.TrimSpace(arguments["id"]),
		Text:     strings.TrimSpace(arguments["text"]),
		Assignee: strings.TrimSpace(arguments["assignee"]),
		Due:      strings.TrimSpace(arguments["due"]),
	}
	if !validName(options.ID, maxTaskIDBytes) {
		return taskOptions{}, fmt.Errorf("task id must be 1-%d bytes and contain only letters, numbers, ., _, -, /, or :", maxTaskIDBytes)
	}
	if options.Text == "" || len(options.Text) > maxTaskTextBytes || !utf8.ValidString(options.Text) {
		return taskOptions{}, fmt.Errorf("task text must be 1-%d bytes of valid UTF-8", maxTaskTextBytes)
	}
	if len(options.Assignee) > maxAssigneeBytes || !utf8.ValidString(options.Assignee) {
		return taskOptions{}, fmt.Errorf("task assignee must be at most %d bytes of valid UTF-8", maxAssigneeBytes)
	}
	if options.Due != "" && !validDueDate(options.Due) {
		return taskOptions{}, fmt.Errorf("task due date must use YYYY-MM-DD")
	}
	switch strings.TrimSpace(arguments["initial"]) {
	case "", "open":
	case "done":
		options.InitialDone = true
	default:
		return taskOptions{}, fmt.Errorf("task initial state must be open or done")
	}
	return options, nil
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
		char := value[index]
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' ||
			char == '-' || char == '_' || char == '.' || char == '/' || char == ':' {
			continue
		}
		return false
	}
	return true
}

// taskDone resolves persisted state and falls back to the declaration's initial state.
func taskDone(options taskOptions, read storageReader) bool {
	if read == nil {
		return options.InitialDone
	}
	stored, err := read(storageKey(options.ID))
	if err != nil || !stored.Found {
		return options.InitialDone
	}
	switch string(stored.Value) {
	case "done":
		return true
	case "open":
		return false
	default:
		return options.InitialDone
	}
}

// renderTask renders one task declaration into safe fallback HTML and browser-module metadata.
func renderTask(options taskOptions, read storageReader) string {
	done := taskDone(options, read)
	action := actionID(options.ID, !done)
	state := "open"
	checked := ""
	if done {
		state = "done"
		checked = " kumbuka-task-done"
	}

	var output strings.Builder
	output.WriteString(`<span class="kumbuka-task-browser" data-kumbuka-plugin="me.kumbuka.tasks" data-kumbuka-module="task-ui" data-kumbuka-input="html">`)
	output.WriteString(`<span class="kumbuka-task-fallback` + checked + `" data-kumbuka-fallback>`)
	output.WriteString(`<span class="kumbuka-task-meta kumbuka-task-action__`)
	output.WriteString(action)
	output.WriteString(` kumbuka-task-state__` + state + `"></span>`)
	output.WriteString(`<span class="kumbuka-task-box" aria-hidden="true">`)
	if done {
		output.WriteString(`✓`)
	}
	output.WriteString(`</span><span class="kumbuka-task-content"><span class="kumbuka-task-text">`)
	output.WriteString(html.EscapeString(options.Text))
	output.WriteString(`</span>`)
	if options.Assignee != "" || options.Due != "" {
		output.WriteString(`<span class="kumbuka-task-details">`)
		if options.Assignee != "" {
			output.WriteString(`<span class="kumbuka-task-assignee">` + html.EscapeString(options.Assignee) + `</span>`)
		}
		if options.Due != "" {
			output.WriteString(`<span class="kumbuka-task-due">Due ` + html.EscapeString(options.Due) + `</span>`)
		}
		output.WriteString(`</span>`)
	}
	output.WriteString(`</span></span></span>`)
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
func actionID(taskID string, done bool) string {
	sum := sha256.Sum256([]byte(taskID))
	state := "open"
	if done {
		state = "done"
	}
	return "task-" + hex.EncodeToString(sum[:12]) + "-" + state
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
