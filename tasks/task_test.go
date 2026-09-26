package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTaskToken verifies required, optional, and invalid task attributes.
func TestParseTaskToken(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="deploy/api" text="Deploy API" assignee="@alice" due="2026-10-01" initial="in-progress"}}`)
		require.NoError(t, err)
		assert.Equal(t, "deploy/api", task.ID)
		assert.Equal(t, "Deploy API", task.Text)
		assert.Equal(t, "@alice", task.Assignee)
		assert.Equal(t, "2026-10-01", task.Due)
		assert.Equal(t, "in-progress", task.InitialState)
	})

	t.Run("invalid assignee", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" assignee="Alice"}}`)
		require.ErrorContains(t, err, "@mention")
	})

	t.Run("default state is deferred to workflow", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="docs" text="Write docs"}}`)
		require.NoError(t, err)
		assert.Empty(t, task.InitialState)
	})

	t.Run("invalid state identifier", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" initial="In progress"}}`)
		require.ErrorContains(t, err, "workflow state ID")
	})

	t.Run("invalid date", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" due="2026-02-31"}}`)
		require.ErrorContains(t, err, "YYYY-MM-DD")
	})

	t.Run("unsupported attribute", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" priority="high"}}`)
		require.ErrorContains(t, err, "unsupported")
	})
}

// TestLoadTaskWorkflow verifies configured ordering, validation, and default behavior.
func TestLoadTaskWorkflow(t *testing.T) {
	t.Run("defaults to open and done", func(t *testing.T) {
		workflow, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) { return sdk.StoredValue{}, nil })
		require.NoError(t, err)
		require.Len(t, workflow.States, 2)
		assert.Equal(t, "open", workflow.States[0].ID)
		assert.Equal(t, "done", workflow.States[1].ID)
		assert.True(t, workflow.States[1].Completed)
	})

	t.Run("preserves configured order", func(t *testing.T) {
		value := []byte(`[{"id":"todo","label":"To do","color":"#64748b","completed":"false"},{"id":"in-progress","label":"In progress","color":"#2563eb","completed":"false"},{"id":"done","label":"Done","color":"#16a34a","completed":"true"}]`)
		workflow, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) {
			return sdk.StoredValue{Found: true, Value: value}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"todo", "in-progress", "done"}, []string{workflow.States[0].ID, workflow.States[1].ID, workflow.States[2].ID})
		assert.Equal(t, "done", workflow.nextState("in-progress").ID)
	})

	t.Run("rejects duplicate IDs", func(t *testing.T) {
		value := []byte(`[{"id":"todo","label":"One","color":"#64748b","completed":"false"},{"id":"todo","label":"Two","color":"#64748b","completed":"false"}]`)
		_, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) {
			return sdk.StoredValue{Found: true, Value: value}, nil
		})
		require.ErrorContains(t, err, "duplicated")
	})
}

// TestTransformSource verifies rendering, stored state, errors, and code preservation.
func TestTransformSource(t *testing.T) {
	workflow := defaultTaskWorkflow()
	read := func(key string) (sdk.StoredValue, error) {
		if key == storageKey("deploy") {
			return sdk.StoredValue{Found: true, Value: []byte("done")}, nil
		}
		return sdk.StoredValue{}, nil
	}

	t.Run("renders legacy done task as configured state", func(t *testing.T) {
		output := transformSource(`{{task id="deploy" text="Deploy API" assignee="@alice" due="2026-10-01"}}`, read, workflow)
		assert.Contains(t, output, "kumbuka-task-completed")
		assert.Contains(t, output, "kumbuka-task-state__done")
		assert.Contains(t, output, actionID("deploy", "open"))
		assert.Contains(t, output, actionID("deploy", "done"))
		assert.Contains(t, output, "Due 2026-10-01")
		assert.Contains(t, output, `<span class="kumbuka-task-assignee" data-kumbuka-mention>@alice</span>`)
	})

	t.Run("renders unknown initial state as error", func(t *testing.T) {
		output := transformSource(`{{task id="deploy" text="Deploy API" initial="review"}}`, nil, workflow)
		assert.Contains(t, output, "Task error:")
		assert.Contains(t, output, "not configured")
	})

	t.Run("renders parse error", func(t *testing.T) {
		output := transformSource(`{{task id="bad id" text="Broken"}}`, nil, workflow)
		assert.Contains(t, output, "Task error:")
	})

	t.Run("preserves code", func(t *testing.T) {
		source := "`{{task id=\"inline\" text=\"Inline\"}}`\n```text\n{{task id=\"fenced\" text=\"Fenced\"}}\n```"
		assert.Equal(t, source, transformSource(source, nil, workflow))
	})
}

// TestDiscoverTasks verifies unique task discovery outside code spans and fences.
func TestDiscoverTasks(t *testing.T) {
	source := "{{task id=\"one\" text=\"One\"}}\n`{{task id=\"inline\" text=\"Inline\"}}`\n{{task id=\"one\" text=\"Duplicate\"}}\n{{task id=\"two\" text=\"Two\"}}"
	tasks := discoverTasks(source)
	require.Len(t, tasks, 2)
	assert.Equal(t, "one", tasks[0].ID)
	assert.Equal(t, "two", tasks[1].ID)
}

// TestResolveAction verifies that only configured states for currently declared tasks are accepted.
func TestResolveAction(t *testing.T) {
	workflow := taskWorkflow{States: []taskWorkflowState{
		{ID: "todo", Label: "To do", Color: "#64748b"},
		{ID: "review", Label: "Review", Color: "#2563eb"},
	}}
	source := `{{task id="deploy" text="Deploy"}}`
	task, state, ok := resolveAction(source, actionID("deploy", "review"), workflow)
	require.True(t, ok)
	assert.Equal(t, "deploy", task.ID)
	assert.Equal(t, "review", state.ID)

	_, _, ok = resolveAction(source, actionID("deploy", "missing"), workflow)
	assert.False(t, ok)
}

// TestApplyTaskAction verifies versioned persistence, configurable states, and notification retries.
func TestApplyTaskAction(t *testing.T) {
	workflow := taskWorkflow{States: []taskWorkflowState{
		{ID: "todo", Label: "To do", Color: "#64748b"},
		{ID: "review", Label: "Review", Color: "#2563eb"},
		{ID: "done", Label: "Done", Color: "#16a34a", Completed: true},
	}}
	page := sdk.Page{Slug: "release/checklist", Title: "Release checklist"}
	source := `{{task id="deploy" text="Deploy API" assignee="@alice" initial="review"}}`
	values := make(map[string][]byte)
	var sent []sdk.NotificationInput
	finalWriteFailure := true
	services := taskMutationServices{
		Read: func(key string) (sdk.StoredValue, error) {
			value, found := values[key]
			return sdk.StoredValue{Value: value, Found: found}, nil
		},
		Write: func(key string, value []byte) error {
			var state taskState
			require.NoError(t, json.Unmarshal(value, &state))
			if state.NotificationSent && finalWriteFailure {
				finalWriteFailure = false
				return errors.New("write failed")
			}
			values[key] = append([]byte(nil), value...)
			return nil
		},
		ResolveMention: func(mention string) (sdk.User, error) {
			assert.Equal(t, "@alice", mention)
			return sdk.User{ID: 42, Mention: mention, DisplayName: "Alice"}, nil
		},
		SendNotification: func(input sdk.NotificationInput) (sdk.Notification, error) {
			sent = append(sent, input)
			return sdk.Notification{ID: 7, RecipientUserID: input.RecipientUserID}, nil
		},
	}

	action := actionID("deploy", "done")
	require.Error(t, applyTaskAction(page, source, action, workflow, services))
	require.NoError(t, applyTaskAction(page, source, action, workflow, services))
	require.Len(t, sent, 2)
	assert.NotEmpty(t, sent[0].IdempotencyKey)
	assert.Equal(t, sent[0].IdempotencyKey, sent[1].IdempotencyKey)
	assert.Equal(t, "Task completed: Deploy API", sent[0].Title)
	assert.Equal(t, sent[0].Title, sent[1].Title)
	assert.Equal(t, int64(42), sent[0].RecipientUserID)
	assert.Equal(t, "/pages/release/checklist", sent[0].URL)

	state := readTaskState(taskOptions{ID: "deploy", InitialState: "review"}, services.Read, workflow)
	assert.Equal(t, "done", state.State)
	assert.True(t, state.NotificationSent)
	assert.Equal(t, uint64(1), state.Version)
}

// TestApplyTaskActionPropagatesReadFailure verifies mutations never overwrite unknown state.
func TestApplyTaskActionPropagatesReadFailure(t *testing.T) {
	failure := errors.New("read failed")
	written := false
	workflow := defaultTaskWorkflow()
	err := applyTaskAction(
		sdk.Page{Slug: "release"},
		`{{task id="deploy" text="Deploy"}}`,
		actionID("deploy", "done"),
		workflow,
		taskMutationServices{
			Read: func(string) (sdk.StoredValue, error) { return sdk.StoredValue{}, failure },
			Write: func(string, []byte) error {
				written = true
				return nil
			},
		},
	)
	assert.ErrorIs(t, err, failure)
	assert.False(t, written)
}

// TestNotifyTaskAssignments verifies assignment notifications are emitted only for new assignees and remain idempotent across retries.
func TestNotifyTaskAssignments(t *testing.T) {
	context := sdk.ContentChangeContext{
		Page: sdk.Page{
			Slug:      "release/checklist",
			Title:     "Release checklist",
			UpdatedAt: time.Date(2026, time.September, 25, 9, 30, 0, 0, time.UTC),
		},
		PreviousSource: `{{task id="existing" text="Existing" assignee="@alice"}}
{{task id="changed" text="Changed" assignee="@alice"}}`,
		Source: `{{task id="existing" text="Existing renamed" assignee="@alice"}}
{{task id="changed" text="Changed" assignee="@bob"}}
{{task id="new" text="New task" assignee="@carol"}}`,
	}
	var resolved []string
	var sent []sdk.NotificationInput
	err := notifyTaskAssignments(context, taskAssignmentServices{
		ResolveMention: func(mention string) (sdk.User, error) {
			resolved = append(resolved, mention)
			return sdk.User{ID: int64(len(resolved) + 40), Mention: mention}, nil
		},
		SendNotification: func(input sdk.NotificationInput) (sdk.Notification, error) {
			sent = append(sent, input)
			return sdk.Notification{ID: int64(len(sent))}, nil
		},
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"@bob", "@carol"}, resolved)
	require.Len(t, sent, 2)
	assert.Equal(t, "Task assigned: Changed", sent[0].Title)
	assert.Equal(t, "/pages/release/checklist", sent[0].URL)
	assert.NotEmpty(t, sent[0].IdempotencyKey)
	assert.NotEqual(t, sent[0].IdempotencyKey, sent[1].IdempotencyKey)

	firstKeys := []string{sent[0].IdempotencyKey, sent[1].IdempotencyKey}
	require.NoError(t, notifyTaskAssignments(context, taskAssignmentServices{
		ResolveMention: func(mention string) (sdk.User, error) {
			return sdk.User{ID: 42, Mention: mention}, nil
		},
		SendNotification: func(input sdk.NotificationInput) (sdk.Notification, error) {
			sent = append(sent, input)
			return sdk.Notification{}, nil
		},
	}))
	require.Len(t, sent, 4)
	assert.Equal(t, firstKeys[0], sent[2].IdempotencyKey)
	assert.Equal(t, firstKeys[1], sent[3].IdempotencyKey)
}

// TestLegacyTaskStateMapsOntoConfiguredWorkflow verifies old boolean persistence keeps its completion meaning.
func TestLegacyTaskStateMapsOntoConfiguredWorkflow(t *testing.T) {
	workflow := taskWorkflow{States: []taskWorkflowState{
		{ID: "todo", Label: "To do", Color: "#64748b"},
		{ID: "complete", Label: "Complete", Color: "#16a34a", Completed: true},
	}}
	legacy := true
	value, err := json.Marshal(taskState{Version: 2, NotificationSent: true, LegacyDone: &legacy})
	require.NoError(t, err)
	state := decodeTaskState(taskOptions{ID: "task"}, sdk.StoredValue{Found: true, Value: value}, workflow)
	assert.Equal(t, "complete", state.State)
	assert.Nil(t, state.LegacyDone)
}

// TestTaskStateNotification verifies notification headings respect state semantics and core byte bounds.
func TestTaskStateNotification(t *testing.T) {
	open := taskWorkflowState{ID: "open", Label: "Open", Color: "#64748b"}
	done := taskWorkflowState{ID: "done", Label: "Done", Color: "#16a34a", Completed: true}
	title, _ := taskStateNotification(open, done, strings.Repeat("é", 200), "Page")
	assert.LessOrEqual(t, len(title), maxTaskNotificationTitleBytes)
	assert.True(t, utf8.ValidString(title))
	assert.True(t, strings.HasPrefix(title, "Task completed: "))
}
