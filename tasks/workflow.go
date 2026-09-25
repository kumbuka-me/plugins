package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	maxTaskStates     = 16
	maxTaskStateID    = 64
	maxTaskStateLabel = 64
)

// settingsReader reads one plugin-owned administrator setting.
type settingsReader func(string) (sdk.StoredValue, error)

// taskWorkflowState describes one administrator-configured task state.
type taskWorkflowState struct {
	// ID is the stable state identifier persisted with task runtime state.
	ID string
	// Label is the human-readable state name.
	Label string
	// Color is the canonical six-digit CSS color used by interactive controls.
	Color string
	// Completed marks states that represent completed work.
	Completed bool
}

// taskWorkflow contains the ordered states available to every task.
type taskWorkflow struct {
	// States contains state definitions in administrator-configured order.
	States []taskWorkflowState
}

// defaultTaskWorkflow returns the backward-compatible Open/Done workflow.
func defaultTaskWorkflow() taskWorkflow {
	return taskWorkflow{States: []taskWorkflowState{
		{ID: "open", Label: "Open", Color: "#64748b"},
		{ID: "done", Label: "Done", Color: "#16a34a", Completed: true},
	}}
}

// loadTaskWorkflow reads the ordered workflow configuration and falls back to Open/Done when no states were saved.
func loadTaskWorkflow(read settingsReader) (taskWorkflow, error) {
	if read == nil {
		return defaultTaskWorkflow(), nil
	}
	stored, err := read("workflow.states")
	if err != nil {
		return taskWorkflow{}, err
	}
	if !stored.Found || len(strings.TrimSpace(string(stored.Value))) == 0 {
		return defaultTaskWorkflow(), nil
	}

	var rows []map[string]string
	if err := json.Unmarshal(stored.Value, &rows); err != nil {
		return taskWorkflow{}, fmt.Errorf("decode task workflow: %w", err)
	}
	if len(rows) == 0 {
		return defaultTaskWorkflow(), nil
	}
	if len(rows) > maxTaskStates {
		return taskWorkflow{}, fmt.Errorf("task workflow allows at most %d states", maxTaskStates)
	}

	workflow := taskWorkflow{States: make([]taskWorkflowState, 0, len(rows))}
	seen := make(map[string]bool, len(rows))
	for index, row := range rows {
		state, err := parseTaskWorkflowState(row)
		if err != nil {
			return taskWorkflow{}, fmt.Errorf("task workflow state %d: %w", index+1, err)
		}
		if seen[state.ID] {
			return taskWorkflow{}, fmt.Errorf("task workflow state %q is duplicated", state.ID)
		}
		seen[state.ID] = true
		workflow.States = append(workflow.States, state)
	}
	return workflow, nil
}

// parseTaskWorkflowState validates one normalized list row from Kumbuka settings.
func parseTaskWorkflowState(row map[string]string) (taskWorkflowState, error) {
	state := taskWorkflowState{
		ID:    strings.TrimSpace(row["id"]),
		Label: strings.TrimSpace(row["label"]),
		Color: strings.ToLower(strings.TrimSpace(row["color"])),
	}
	if !validTaskStateID(state.ID) {
		return taskWorkflowState{}, fmt.Errorf("state ID must be 1-%d lower-case letters, numbers, dots, underscores, or hyphens", maxTaskStateID)
	}
	if state.Label == "" || len(state.Label) > maxTaskStateLabel || !utf8.ValidString(state.Label) {
		return taskWorkflowState{}, fmt.Errorf("state label must be 1-%d bytes of valid UTF-8", maxTaskStateLabel)
	}
	if !validTaskStateColor(state.Color) {
		return taskWorkflowState{}, fmt.Errorf("state color must be a six-digit hex color")
	}
	switch strings.TrimSpace(row["completed"]) {
	case "", "false":
	case "true":
		state.Completed = true
	default:
		return taskWorkflowState{}, fmt.Errorf("completed must be true or false")
	}
	return state, nil
}

// validTaskStateID reports whether value is safe for persisted state and browser metadata classes.
func validTaskStateID(value string) bool {
	if value == "" || len(value) > maxTaskStateID || !taskStateIDByte(value[0], true) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !taskStateIDByte(value[index], false) {
			return false
		}
	}
	return true
}

// taskStateIDByte reports whether character is allowed at one position in a workflow state ID.
func taskStateIDByte(character byte, first bool) bool {
	if asciiLowerLetterOrDigit(character) {
		return true
	}
	return !first && (character == '-' || character == '_' || character == '.')
}

// asciiLowerLetterOrDigit reports whether character is an ASCII lower-case letter or decimal digit.
func asciiLowerLetterOrDigit(character byte) bool {
	return character >= 'a' && character <= 'z' || character >= '0' && character <= '9'
}

// validTaskStateColor reports whether value is a canonical six-digit CSS hex color.
func validTaskStateColor(value string) bool {
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !asciiLowerHexDigit(value[index]) {
			return false
		}
	}
	return true
}

// asciiLowerHexDigit reports whether character is a lower-case ASCII hexadecimal digit.
func asciiLowerHexDigit(character byte) bool {
	return character >= '0' && character <= '9' || character >= 'a' && character <= 'f'
}

// state returns one configured state by ID.
func (w taskWorkflow) state(id string) (taskWorkflowState, bool) {
	for _, state := range w.States {
		if state.ID == id {
			return state, true
		}
	}
	return taskWorkflowState{}, false
}

// initialState resolves an explicit declaration state or the first configured workflow state.
func (w taskWorkflow) initialState(id string) (taskWorkflowState, error) {
	if len(w.States) == 0 {
		return taskWorkflowState{}, fmt.Errorf("task workflow has no states")
	}
	if id == "" {
		return w.States[0], nil
	}
	state, ok := w.state(id)
	if !ok {
		return taskWorkflowState{}, fmt.Errorf("task state %q is not configured", id)
	}
	return state, nil
}

// legacyState maps the old boolean completion model onto the closest configured state.
func (w taskWorkflow) legacyState(done bool) taskWorkflowState {
	preferred := "open"
	if done {
		preferred = "done"
	}
	if state, ok := w.state(preferred); ok {
		return state
	}
	for _, state := range w.States {
		if state.Completed == done {
			return state
		}
	}
	if len(w.States) > 0 {
		return w.States[0]
	}
	return defaultTaskWorkflow().legacyState(done)
}

// nextState returns the state following current, wrapping at the end of the workflow.
func (w taskWorkflow) nextState(current string) taskWorkflowState {
	if len(w.States) == 0 {
		return defaultTaskWorkflow().States[0]
	}
	for index, state := range w.States {
		if state.ID == current {
			return w.States[(index+1)%len(w.States)]
		}
	}
	return w.States[0]
}
