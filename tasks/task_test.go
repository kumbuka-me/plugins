package main

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
)

// TestParseTaskToken verifies required, optional, and invalid task attributes.
func TestParseTaskToken(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="deploy/api" text="Deploy API" assignee="@alice" due="2026-10-01" initial="done"}}`)
		if err != nil {
			t.Fatalf("parseTaskToken() error = %v", err)
		}
		if task.ID != "deploy/api" || task.Text != "Deploy API" || task.Assignee != "@alice" || task.Due != "2026-10-01" || !task.InitialDone {
			t.Fatalf("unexpected task: %+v", task)
		}
	})

	t.Run("invalid assignee", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" assignee="Alice"}}`)
		if err == nil || !strings.Contains(err.Error(), "@mention") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("defaults open", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="docs" text="Write docs"}}`)
		if err != nil || task.InitialDone {
			t.Fatalf("unexpected task=%+v error=%v", task, err)
		}
	})

	t.Run("invalid date", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" due="2026-02-31"}}`)
		if err == nil || !strings.Contains(err.Error(), "YYYY-MM-DD") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("unsupported attribute", func(t *testing.T) {
		_, err := parseTaskToken(`{{task id="docs" text="Write docs" priority="high"}}`)
		if err == nil || !strings.Contains(err.Error(), "unsupported") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestTransformSource verifies rendering, stored state, errors, and code preservation.
func TestTransformSource(t *testing.T) {
	read := func(key string) (sdk.StoredValue, error) {
		if key == storageKey("deploy") {
			return sdk.StoredValue{Found: true, Value: []byte("done")}, nil
		}
		return sdk.StoredValue{}, nil
	}

	t.Run("renders stored done task", func(t *testing.T) {
		output := transformSource(`{{task id="deploy" text="Deploy API" assignee="@alice" due="2026-10-01"}}`, read)
		if !strings.Contains(output, "kumbuka-task-done") || !strings.Contains(output, "Deploy API") || !strings.Contains(output, "Due 2026-10-01") {
			t.Fatalf("task was not rendered: %s", output)
		}
		if !strings.Contains(output, actionID("deploy", false)) {
			t.Fatalf("reopen action missing: %s", output)
		}
	})

	t.Run("renders error", func(t *testing.T) {
		output := transformSource(`{{task id="bad id" text="Broken"}}`, nil)
		if !strings.Contains(output, "Task error:") {
			t.Fatalf("task error missing: %s", output)
		}
	})

	t.Run("preserves code", func(t *testing.T) {
		source := "`{{task id=\"inline\" text=\"Inline\"}}`\n```text\n{{task id=\"fenced\" text=\"Fenced\"}}\n```"
		if output := transformSource(source, nil); output != source {
			t.Fatalf("code changed:\n%s", output)
		}
	})
}

// TestDiscoverTasks verifies unique task discovery outside code spans and fences.
func TestDiscoverTasks(t *testing.T) {
	source := "{{task id=\"one\" text=\"One\"}}\n`{{task id=\"inline\" text=\"Inline\"}}`\n{{task id=\"one\" text=\"Duplicate\"}}\n{{task id=\"two\" text=\"Two\"}}"
	tasks := discoverTasks(source)
	if len(tasks) != 2 || tasks[0].ID != "one" || tasks[1].ID != "two" {
		t.Fatalf("unexpected tasks: %+v", tasks)
	}
}

// TestResolveAction verifies that only actions for currently declared tasks are accepted.
func TestResolveAction(t *testing.T) {
	source := `{{task id="deploy" text="Deploy"}}`
	task, done, ok := resolveAction(source, actionID("deploy", true))
	if !ok || task.ID != "deploy" || !done {
		t.Fatalf("unexpected resolved action: task=%+v done=%v ok=%v", task, done, ok)
	}
	if _, _, ok := resolveAction(source, actionID("missing", true)); ok {
		t.Fatal("unexpected action acceptance")
	}
}

// TestApplyTaskAction verifies versioned persistence, user resolution, and notification retries.
func TestApplyTaskAction(t *testing.T) {
	page := sdk.Page{Slug: "release/checklist", Title: "Release checklist"}
	source := `{{task id="deploy" text="Deploy API" assignee="@alice"}}`
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
			if err := json.Unmarshal(value, &state); err != nil {
				return err
			}
			if state.NotificationSent && finalWriteFailure {
				finalWriteFailure = false
				return errors.New("write failed")
			}
			values[key] = append([]byte(nil), value...)
			return nil
		},
		ResolveMention: func(mention string) (sdk.User, error) {
			if mention != "@alice" {
				t.Fatalf("unexpected mention: %s", mention)
			}
			return sdk.User{ID: 42, Mention: mention, DisplayName: "Alice"}, nil
		},
		SendNotification: func(input sdk.NotificationInput) (sdk.Notification, error) {
			sent = append(sent, input)
			return sdk.Notification{ID: 7, RecipientUserID: input.RecipientUserID}, nil
		},
	}

	action := actionID("deploy", true)
	if err := applyTaskAction(page, source, action, services); err == nil {
		t.Fatal("expected final state write failure")
	}
	if err := applyTaskAction(page, source, action, services); err != nil {
		t.Fatalf("retry task action: %v", err)
	}
	if len(sent) != 2 {
		t.Fatalf("notifications sent = %d, want 2 idempotent attempts", len(sent))
	}
	if sent[0].IdempotencyKey == "" || sent[0].IdempotencyKey != sent[1].IdempotencyKey {
		t.Fatalf("idempotency keys differ: %+v", sent)
	}
	if sent[0].RecipientUserID != 42 || sent[0].URL != "/pages/release/checklist" {
		t.Fatalf("unexpected notification: %+v", sent[0])
	}
	state := readTaskState(taskOptions{ID: "deploy"}, services.Read)
	if !state.Done || !state.NotificationSent || state.Version != 1 {
		t.Fatalf("unexpected final state: %+v", state)
	}
}

// TestApplyTaskActionPropagatesReadFailure verifies mutations never overwrite unknown state.
func TestApplyTaskActionPropagatesReadFailure(t *testing.T) {
	failure := errors.New("read failed")
	written := false
	err := applyTaskAction(
		sdk.Page{Slug: "release"},
		`{{task id="deploy" text="Deploy"}}`,
		actionID("deploy", true),
		taskMutationServices{
			Read: func(string) (sdk.StoredValue, error) { return sdk.StoredValue{}, failure },
			Write: func(string, []byte) error {
				written = true
				return nil
			},
		},
	)
	if !errors.Is(err, failure) || written {
		t.Fatalf("unexpected mutation result: err=%v written=%v", err, written)
	}
}

// TestTaskDone verifies persisted state takes precedence over declaration defaults.
func TestTaskDone(t *testing.T) {
	options := taskOptions{ID: "task", InitialDone: true}
	read := func(string) (sdk.StoredValue, error) {
		return sdk.StoredValue{Found: true, Value: []byte("open")}, nil
	}
	if taskDone(options, read) {
		t.Fatal("stored open state did not override initial done")
	}
}

// TestTaskNotificationTitle verifies notification headings respect core byte bounds.
func TestTaskNotificationTitle(t *testing.T) {
	title := taskNotificationTitle(true, strings.Repeat("é", 200))
	if len(title) > maxTaskNotificationTitleBytes || !utf8.ValidString(title) {
		t.Fatalf("invalid bounded title: %q (%d bytes)", title, len(title))
	}
}
