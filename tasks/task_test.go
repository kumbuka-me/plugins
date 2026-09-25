package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

// TestParseTaskToken verifies required, optional, and invalid task attributes.
func TestParseTaskToken(t *testing.T) {
	t.Run("complete", func(t *testing.T) {
		task, err := parseTaskToken(`{{task id="deploy/api" text="Deploy API" assignee="Platform" due="2026-10-01" initial="done"}}`)
		if err != nil {
			t.Fatalf("parseTaskToken() error = %v", err)
		}
		if task.ID != "deploy/api" || task.Text != "Deploy API" || task.Assignee != "Platform" || task.Due != "2026-10-01" || !task.InitialDone {
			t.Fatalf("unexpected task: %+v", task)
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
		output := transformSource(`{{task id="deploy" text="Deploy API" assignee="Platform" due="2026-10-01"}}`, read)
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
	id, done, ok := resolveAction(source, actionID("deploy", true))
	if !ok || id != "deploy" || !done {
		t.Fatalf("unexpected resolved action: id=%q done=%v ok=%v", id, done, ok)
	}
	if _, _, ok := resolveAction(source, actionID("missing", true)); ok {
		t.Fatal("unexpected action acceptance")
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
