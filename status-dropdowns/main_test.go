package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

// TestParseStatusToken verifies supported declaration attributes.
func TestParseStatusToken(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		options, ok := parseStatusToken(`{{status id="release/api" set="workflow" initial="In progress" prefix="API" style="outline"}}`)
		if !ok {
			t.Fatal("expected declaration to parse")
		}
		if options.ID != "release/api" || options.Set != "workflow" || options.Initial != "In progress" || options.Prefix != "API" || options.Style != "outline" {
			t.Fatalf("unexpected options: %+v", options)
		}
	})

	t.Run("unknown attribute", func(t *testing.T) {
		if _, ok := parseStatusToken(`{{status id="release" set="workflow" color="red"}}`); ok {
			t.Fatal("expected unknown attribute to be rejected")
		}
	})

	t.Run("invalid identifier", func(t *testing.T) {
		if _, ok := parseStatusToken(`{{status id="release api" set="workflow"}}`); ok {
			t.Fatal("expected invalid identifier to be rejected")
		}
	})
}

// TestParseStatusSet verifies custom set parsing and validation.
func TestParseStatusSet(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		set, ok := parseStatusSet("delivery", "Planned|gray\nDeploying|blue\nLive|green")
		if !ok || len(set.Choices) != 3 || set.Choices[1].Label != "Deploying" || set.Choices[1].Tone != "blue" {
			t.Fatalf("unexpected set: %+v", set)
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		if _, ok := parseStatusSet("bad", "Done|green\nDone|blue"); ok {
			t.Fatal("expected duplicate labels to be rejected")
		}
	})

	t.Run("unknown tone", func(t *testing.T) {
		if _, ok := parseStatusSet("bad", "Done|rainbow"); ok {
			t.Fatal("expected unknown tone to be rejected")
		}
	})
}

// TestTransformSource verifies inline rendering, table support, stored values, and code preservation.
func TestTransformSource(t *testing.T) {
	readResource := func(string, string) (sdk.PluginResourceRecord, error) {
		return sdk.PluginResourceRecord{}, errTestMissing
	}
	readStorage := func(key string) (sdk.StoredValue, error) {
		if key == storageKey("release/api") {
			return sdk.StoredValue{Found: true, Value: []byte("Done")}, nil
		}
		return sdk.StoredValue{}, nil
	}

	source := "| API | {{status id=\"release/api\" set=\"workflow\" prefix=\"API\"}} |\n\n`{{status id=\"inline-code\" set=\"workflow\"}}`\n\n```text\n{{status id=\"fenced\" set=\"workflow\"}}\n```"
	output := transformSource(source, readResource, readStorage)
	if !strings.Contains(output, "kumbuka-status-green") || !strings.Contains(output, ">Done</span>") {
		t.Fatalf("stored status was not rendered: %s", output)
	}
	if !strings.Contains(output, "`{{status id=\"inline-code\" set=\"workflow\"}}`") {
		t.Fatalf("inline code changed: %s", output)
	}
	if !strings.Contains(output, "{{status id=\"fenced\" set=\"workflow\"}}") {
		t.Fatalf("fenced code changed: %s", output)
	}
}

// TestDiscoverStatuses verifies page controls ignore code and deduplicate persistent IDs.
func TestDiscoverStatuses(t *testing.T) {
	source := "A {{status id=\"one\" set=\"workflow\"}}\nB {{status id=\"one\" set=\"workflow\"}}\n`{{status id=\"inline\" set=\"workflow\"}}`\n```\n{{status id=\"fenced\" set=\"workflow\"}}\n```\n{{status id=\"two\" set=\"approval\"}}"
	got := discoverStatuses(source)
	if len(got) != 2 || got[0].ID != "one" || got[1].ID != "two" {
		t.Fatalf("unexpected declarations: %+v", got)
	}
}

// TestResolveAction verifies commands are derived from current page declarations and configured choices.
func TestResolveAction(t *testing.T) {
	source := `{{status id="release" set="workflow"}}`
	action := actionID("release", 3)
	id, value, ok := resolveAction(source, action, func(string, string) (sdk.PluginResourceRecord, error) {
		return sdk.PluginResourceRecord{}, errTestMissing
	})
	if !ok || id != "release" || value != "Done" {
		t.Fatalf("unexpected action resolution: id=%q value=%q ok=%v", id, value, ok)
	}
}

// testError provides a dependency-free sentinel error for resource fallbacks.
type testError string

// Error returns the sentinel error text.
func (e testError) Error() string { return string(e) }

const errTestMissing = testError("missing")
