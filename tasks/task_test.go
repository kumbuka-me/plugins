package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseTaskToken verifies required, optional, and invalid task attributes.
func TestParseTaskToken(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="deploy/api" text="Deploy API" description="Roll out the API after smoke tests." assignee="@alice" due="2026-10-01" initial="in-progress" parent="release" workflow="release-flow"}}`)
		require.NoError(t, err)
		assert.Equal(t, "deploy/api", task.ID)
		assert.Equal(t, "Deploy API", task.Text)
		assert.Equal(t, "Roll out the API after smoke tests.", task.Description)
		assert.Equal(t, "@alice", task.Assignee)
		assert.Equal(t, "2026-10-01", task.Due)
		assert.Equal(t, "in-progress", task.InitialState)
		assert.Equal(t, "release", task.Parent)
		assert.Equal(t, "release-flow", task.Workflow)
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

	t.Run("rejects self parent", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" parent="docs"}}`)
		require.ErrorContains(t, err, "own parent")
	})
}

// TestParseTaskListToken verifies one visual-editor task list expands into a validated hierarchy.
func TestParseTaskListToken(t *testing.T) {
	separator := `\u001f`
	token := `{{tasks texts="Prepare release` + separator + `Publish notes` + separator + `Verify production" descriptions="Release everything` + separator + `Write notes` + separator + `Check health" ids="release` + separator + `notes` + separator + `verify" parents="` + separator + `release` + separator + `notes" assignees="@alice` + separator + `@alice` + separator + `" dues="2026-10-01` + separator + separator + `2026-10-02"}}`

	tasks, err := parseTaskListToken(token)
	require.NoError(t, err)
	require.Len(t, tasks, 3)
	assert.Equal(t, "release", tasks[0].ID)
	assert.Empty(t, tasks[0].Parent)
	assert.Equal(t, "release", tasks[1].Parent)
	assert.Equal(t, "Write notes", tasks[1].Description)
	assert.Equal(t, "notes", tasks[2].Parent)
	assert.Equal(t, "@alice", tasks[1].Assignee)
	assert.Equal(t, "2026-10-02", tasks[2].Due)
}

func TestTaskListWorkflowGroup(t *testing.T) {
	tasks, err := parseTaskListToken(`{{tasks workflow="release-flow" texts="Prepare release" ids="release" initials="review"}}`)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	require.Equal(t, "release-flow", tasks[0].Workflow)

	readWorkflow := func(resource, key string) (sdk.PluginResourceRecord, error) {
		require.Equal(t, "workflows", resource)
		require.Equal(t, "release-flow", key)
		return sdk.PluginResourceRecord{Key: key, Values: map[string]string{
			"states": `[{"id":"draft","label":"Draft","color":"#64748b","completed":"false"},{"id":"review","label":"Review","color":"#2563eb","completed":"false"},{"id":"shipped","label":"Shipped","color":"#16a34a","completed":"true"}]`,
		}}, nil
	}
	output := transformSource(`{{tasks workflow="release-flow" texts="Prepare release" ids="release" initials="review"}}`, nil, defaultTaskWorkflow(), readWorkflow, localize.For("en"))
	require.Contains(t, output, "kumbuka-task-state__review")
	require.Contains(t, output, actionID("release", "shipped"))
}

// TestParseTaskListTokenGeneratesMissingIDs verifies a freshly inserted visual task is renderable before its first Apply.
func TestParseTaskListTokenGeneratesMissingIDs(t *testing.T) {
	tasks, err := parseTaskListToken(`{{tasks texts="Describe the task"}}`)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "Describe the task", tasks[0].Text)
	assert.Regexp(t, `^task-[0-9a-f]{24}$`, tasks[0].ID)
}

// TestParseTaskListTokenRejectsForwardParent verifies modal rows form a deterministic document-order tree.
func TestParseTaskListTokenRejectsForwardParent(t *testing.T) {
	separator := `\u001f`
	_, err := parseTaskListToken(`{{tasks texts="Child` + separator + `Parent" ids="child` + separator + `parent" parents="parent` + separator + `"}}`)
	require.ErrorContains(t, err, "must appear before")
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
		value := []byte(`[{"id":"todo","label":"To do","color":"#64748b","completed":"false","default":"false"},{"id":"in-progress","label":"In progress","color":"#2563eb","completed":"false","default":"true"},{"id":"done","label":"Done","color":"#16a34a","completed":"true","default":"false"}]`)
		workflow, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) {
			return sdk.StoredValue{Found: true, Value: value}, nil
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"todo", "in-progress", "done"}, []string{workflow.States[0].ID, workflow.States[1].ID, workflow.States[2].ID})
		assert.Equal(t, "done", workflow.nextState("in-progress").ID)
		initial, initialErr := workflow.initialState("")
		require.NoError(t, initialErr)
		assert.Equal(t, "in-progress", initial.ID)
	})

	t.Run("rejects multiple defaults", func(t *testing.T) {
		value := []byte(`[{"id":"todo","label":"To do","color":"#64748b","completed":"false","default":"true"},{"id":"review","label":"Review","color":"#2563eb","completed":"false","default":"true"}]`)
		_, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) {
			return sdk.StoredValue{Found: true, Value: value}, nil
		})
		require.ErrorContains(t, err, "more than one default")
	})

	t.Run("rejects duplicate IDs", func(t *testing.T) {
		value := []byte(`[{"id":"todo","label":"One","color":"#64748b","completed":"false"},{"id":"todo","label":"Two","color":"#64748b","completed":"false"}]`)
		_, err := loadTaskWorkflow(func(string) (sdk.StoredValue, error) {
			return sdk.StoredValue{Found: true, Value: value}, nil
		})
		require.ErrorContains(t, err, "duplicated")
	})
}

func TestTaskWorkflowFromRecordRejectsDuplicateStates(t *testing.T) {
	_, err := taskWorkflowFromRecord("release", sdk.PluginResourceRecord{Key: "release", Values: map[string]string{
		"states": `[{"id":"open","label":"Open","color":"#64748b","completed":"false"},{"id":"open","label":"Again","color":"#64748b","completed":"false"}]`,
	}})
	require.ErrorContains(t, err, "duplicated")
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
		output := transformSource(`{{task id="deploy" text="Deploy API" description="Deploy after verification." assignee="@alice" due="2026-10-01"}}`, read, workflow, nil, localize.For("en"))
		assert.Contains(t, output, "kumbuka-task-completed")
		assert.Contains(t, output, "kumbuka-task-state__done")
		assert.Contains(t, output, actionID("deploy", "open"))
		assert.Contains(t, output, actionID("deploy", "done"))
		assert.Contains(t, output, "Due 2026-10-01")
		assert.Contains(t, output, `<span class="kumbuka-task-description">Deploy after verification.</span>`)
		assert.Contains(t, output, `<span class="kumbuka-task-assignee" data-kumbuka-mention>@alice</span>`)
	})

	t.Run("renders unknown initial state as error", func(t *testing.T) {
		output := transformSource(`{{task id="deploy" text="Deploy API" initial="review"}}`, nil, workflow, nil, localize.For("en"))
		assert.Contains(t, output, "Task error:")
		assert.Contains(t, output, "not configured")
	})

	t.Run("renders parse error", func(t *testing.T) {
		output := transformSource(`{{task id="bad id" text="Broken"}}`, nil, workflow, nil, localize.For("en"))
		assert.Contains(t, output, "Task error:")
	})

	t.Run("renders visual editor task list", func(t *testing.T) {
		separator := `\u001f`
		source := `{{tasks texts="Prepare release` + separator + `Publish notes` + separator + `Verify production" ids="release` + separator + `notes` + separator + `verify" parents="` + separator + `release` + separator + `notes"}}`
		output := transformSource(source, nil, workflow, nil, localize.For("en"))
		assert.Equal(t, 1, strings.Count(output, `class="kumbuka-task-browser"`))
		assert.Contains(t, output, "kumbuka-task-depth__2")
		assert.Contains(t, output, "kumbuka-task-children")
		assert.Contains(t, output, "Verify production")
		assert.NotContains(t, output, "kumbuka-task-choices")
	})

	t.Run("preserves code", func(t *testing.T) {
		source := "`{{task id=\"inline\" text=\"Inline\"}}`\n```text\n{{task id=\"fenced\" text=\"Fenced\"}}\n```"
		assert.Equal(t, source, transformSource(source, nil, workflow, nil, localize.For("en")))
	})
}

// TestRenderTaskListHeader verifies the progress count is emitted directly into the rendered header.
func TestRenderTaskListHeader(t *testing.T) {
	header := renderTaskListHeader("Tasks", 1, 5)
	assert.Equal(t, 1, strings.Count(header, `class="kumbuka-task-progress-current">1</span>`))
	assert.Contains(t, header, `class="kumbuka-task-progress-total">5</span>`)
}

// TestTaskListRendering verifies adjacent tasks are grouped and parent references create compact nesting.
func TestTaskListRendering(t *testing.T) {
	workflow := defaultTaskWorkflow()
	read := func(key string) (sdk.StoredValue, error) {
		if key == storageKey("release-notes") {
			return sdk.StoredValue{Found: true, Value: []byte("done")}, nil
		}
		return sdk.StoredValue{}, nil
	}
	source := `{{task id="release" text="Prepare release"}}

{{task id="release-notes" parent="release" text="Publish release notes"}}

{{task id="production" parent="release" text="Deploy to production"}}

{{task id="verify" parent="production" text="Verify production health"}}

{{task id="announce" text="Announce the release"}}`

	output := transformSource(source, read, workflow, nil, localize.For("en"))
	assert.Equal(t, 1, strings.Count(output, `class="kumbuka-task-browser"`))
	assert.Contains(t, output, `class="kumbuka-task-list-header"`)
	assert.Contains(t, output, `class="kumbuka-task-progress-current">1</span>`)
	assert.Contains(t, output, "kumbuka-task-depth__0")
	assert.Contains(t, output, "kumbuka-task-depth__1")
	assert.Contains(t, output, "kumbuka-task-depth__2")
	assert.Contains(t, output, "Publish release notes")
}

// TestTaskListInvalidParent verifies subtasks cannot point outside their adjacent task list.
func TestTaskListInvalidParent(t *testing.T) {
	workflow := defaultTaskWorkflow()
	output := transformSource(`{{task id="release" text="Prepare release"}}
{{task id="verify" parent="missing" text="Verify"}}`, nil, workflow, nil, localize.For("en"))
	require.Contains(t, output, "Task error:")
	assert.Contains(t, output, "must appear before its subtask")
}

// TestDiscoverTasks verifies unique task discovery outside code spans and fences.
func TestDiscoverTasks(t *testing.T) {
	separator := `\u001f`
	source := "{{task id=\"one\" text=\"One\"}}\n`{{task id=\"inline\" text=\"Inline\"}}`\n{{task id=\"one\" text=\"Duplicate\"}}\n" +
		`{{tasks texts="Two` + separator + `Three" ids="two` + separator + `three" parents="` + separator + `two"}}`
	tasks := discoverTasks(source)
	require.Len(t, tasks, 3)
	assert.Equal(t, "one", tasks[0].ID)
	assert.Equal(t, "two", tasks[1].ID)
	assert.Equal(t, "three", tasks[2].ID)
	assert.Equal(t, "two", tasks[2].Parent)
}

// TestResolveAction verifies that only configured states for currently declared tasks are accepted.
func TestResolveAction(t *testing.T) {
	workflow := taskWorkflow{States: []taskWorkflowState{
		{ID: "todo", Label: "To do", Color: "#64748b"},
		{ID: "review", Label: "Review", Color: "#2563eb"},
	}}
	source := `{{task id="deploy" text="Deploy"}}`
	task, state, ok := resolveAction(source, actionID("deploy", "review"), workflow, nil)
	require.True(t, ok)
	assert.Equal(t, "deploy", task.ID)
	assert.Equal(t, "review", state.ID)

	_, _, ok = resolveAction(source, actionID("deploy", "missing"), workflow, nil)
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
	require.Error(t, applyTaskAction(page, source, action, workflow, nil, services, localize.For("en")))
	require.NoError(t, applyTaskAction(page, source, action, workflow, nil, services, localize.For("en")))
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
		nil,
		taskMutationServices{
			Read: func(string) (sdk.StoredValue, error) { return sdk.StoredValue{}, failure },
			Write: func(string, []byte) error {
				written = true
				return nil
			},
		},
		localize.For("en"),
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
	}, localize.For("en"))

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
	}, localize.For("en")))
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
	title, _ := taskStateNotification(open, done, strings.Repeat("é", 200), "Page", "Done", localize.For("en"))
	assert.LessOrEqual(t, len(title), maxTaskNotificationTitleBytes)
	assert.True(t, utf8.ValidString(title))
	assert.True(t, strings.HasPrefix(title, "Task completed: "))
}

func TestTaskPresentationAndNotificationsGerman(t *testing.T) {
	workflow := defaultTaskWorkflow()
	output := transformSource(`{{task id="deploy" text="Deploy" due="2026-10-01"}}`, nil, workflow, nil, localize.For("de"))
	require.Contains(t, output, "Offen")
	require.Contains(t, output, "Fällig 2026-10-01")

	title, body := taskStateNotification(workflow.States[0], workflow.States[1], "Deploy", "Release", workflow.stateLabel(workflow.States[1], localize.For("de")), localize.For("de"))
	assert.Equal(t, "Aufgabe abgeschlossen: Deploy", title)
	assert.Equal(t, "Eine Aufgabe auf Release wurde als Erledigt abgeschlossen.", body)
}
