package main

import (
	"encoding/hex"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

// TestParseStatusToken verifies reusable and page-local declaration attributes.
func TestParseStatusToken(t *testing.T) {
	t.Run("reusable set", func(t *testing.T) {
		options, err := parseStatusToken(`{{status id="release/api" set="workflow" initial="In progress" prefix="API" style="outline"}}`)
		if err != nil {
			t.Fatalf("parseStatusToken() error = %v", err)
		}
		if options.ID != "release/api" || options.Set != "workflow" || options.Initial != "In progress" || options.Prefix != "API" || options.Style != "outline" {
			t.Fatalf("unexpected options: %+v", options)
		}
	})

	t.Run("page local options", func(t *testing.T) {
		options, err := parseStatusToken(`{{status id="risk" options="Low;Medium;High" colors="gray;#2563eb;red"}}`)
		if err != nil {
			t.Fatalf("parseStatusToken() error = %v", err)
		}
		if options.Set != "" || len(options.Choices) != 3 || options.Choices[1].Color != "#2563eb" || options.Choices[2].Color != "#dc2626" {
			t.Fatalf("unexpected local choices: %+v", options)
		}
	})

	t.Run("default page local colors", func(t *testing.T) {
		options, err := parseStatusToken(`{{status id="risk" options="Low, Medium, High"}}`)
		if err != nil {
			t.Fatalf("parseStatusToken() error = %v", err)
		}
		if len(options.Choices) != 3 || options.Choices[0].Color != "#64748b" || options.Choices[1].Color != "#2563eb" {
			t.Fatalf("unexpected default colors: %+v", options.Choices)
		}
	})

	t.Run("set and options conflict", func(t *testing.T) {
		_, err := parseStatusToken(`{{status id="release" set="workflow" options="One;Two"}}`)
		if err == nil || !strings.Contains(err.Error(), "either set or options") {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid identifier", func(t *testing.T) {
		_, err := parseStatusToken(`{{status id="release api" set="workflow"}}`)
		if err == nil || !strings.Contains(err.Error(), "status id") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestStatusSetFromRecord verifies structured reusable sets and legacy compatibility.
func TestStatusSetFromRecord(t *testing.T) {
	t.Run("structured", func(t *testing.T) {
		set, err := statusSetFromRecord("delivery", sdk.PluginResourceRecord{
			Key: "delivery",
			Values: map[string]string{
				"statuses": `[{"label":"Planned","color":"#64748b"},{"label":"Live","color":"#16a34a"}]`,
			},
		})
		if err != nil || len(set.Choices) != 2 || set.Choices[1].Label != "Live" || set.Choices[1].Color != "#16a34a" {
			t.Fatalf("unexpected set: %+v error=%v", set, err)
		}
	})

	t.Run("legacy", func(t *testing.T) {
		set, err := statusSetFromRecord("delivery", sdk.PluginResourceRecord{
			Key:    "delivery",
			Values: map[string]string{"statuses": "Planned|gray\nLive|green"},
		})
		if err != nil || len(set.Choices) != 2 || set.Choices[1].Color != "#16a34a" {
			t.Fatalf("unexpected legacy set: %+v error=%v", set, err)
		}
	})

	t.Run("reports invalid legacy color", func(t *testing.T) {
		_, err := statusSetFromRecord("delivery", sdk.PluginResourceRecord{
			Key:    "delivery",
			Values: map[string]string{"statuses": "Planned|gray\nBroken|rosa"},
		})
		if err == nil || !strings.Contains(err.Error(), `invalid color "rosa"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

// TestTransformSource verifies interactive rendering, errors, stored values, and code preservation.
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

	t.Run("renders reusable interactive status", func(t *testing.T) {
		source := `| API | {{status id="release/api" set="workflow" prefix="API"}} |`
		output := transformSource(source, readResource, readStorage)
		encodedColor := hex.EncodeToString([]byte("#16a34a"))
		if !strings.Contains(output, "kumbuka-status-green") || !strings.Contains(output, ">Done</span>") {
			t.Fatalf("stored status was not rendered: %s", output)
		}
		if !strings.Contains(output, `data-kumbuka-plugin="me.kumbuka.status-dropdowns"`) ||
			!strings.Contains(output, `data-kumbuka-module="status-ui"`) ||
			!strings.Contains(output, "kumbuka-status-choice__"+encodedColor+"__"+actionID("release/api", 3)+"__446f6e65") {
			t.Fatalf("interactive status metadata was not rendered: %s", output)
		}
	})

	t.Run("renders page local status", func(t *testing.T) {
		output := transformSource(`{{status id="risk" options="Low;High" colors="#64748b;#dc2626" initial="High"}}`, readResource, nil)
		if !strings.Contains(output, ">High</span>") || !strings.Contains(output, hex.EncodeToString([]byte("#dc2626"))) {
			t.Fatalf("page-local status was not rendered: %s", output)
		}
	})

	t.Run("renders declaration error", func(t *testing.T) {
		output := transformSource(`{{status id="risk" set="workflow" options="Low;High"}}`, readResource, nil)
		if !strings.Contains(output, "Status error:") || !strings.Contains(output, "either set or options") {
			t.Fatalf("status error was not rendered: %s", output)
		}
	})

	t.Run("renders set error", func(t *testing.T) {
		readInvalid := func(string, string) (sdk.PluginResourceRecord, error) {
			return sdk.PluginResourceRecord{Key: "broken", Values: map[string]string{"statuses": `[{"label":"Ready","color":"nope"}]`}}, nil
		}
		output := transformSource(`{{status id="broken" set="broken"}}`, readInvalid, nil)
		if !strings.Contains(output, `Status error: invalid status set &#34;broken&#34;`) || !strings.Contains(output, "invalid color") {
			t.Fatalf("status-set error was not rendered: %s", output)
		}
	})

	t.Run("preserves code", func(t *testing.T) {
		source := "`{{status id=\"inline-code\" set=\"workflow\"}}`\n\n```text\n{{status id=\"fenced\" set=\"workflow\"}}\n```"
		output := transformSource(source, readResource, readStorage)
		if !strings.Contains(output, "`{{status id=\"inline-code\" set=\"workflow\"}}`") {
			t.Fatalf("inline code changed: %s", output)
		}
		if !strings.Contains(output, "{{status id=\"fenced\" set=\"workflow\"}}") {
			t.Fatalf("fenced code changed: %s", output)
		}
	})
}

// TestDiscoverStatuses verifies page controls ignore code and deduplicate persistent IDs.
func TestDiscoverStatuses(t *testing.T) {
	source := "A {{status id=\"one\" options=\"Open;Closed\"}}\nB {{status id=\"one\" options=\"Open;Closed\"}}\n`{{status id=\"inline\" set=\"workflow\"}}`\n```\n{{status id=\"fenced\" set=\"workflow\"}}\n```\n{{status id=\"two\" set=\"approval\"}}"
	got := discoverStatuses(source)
	if len(got) != 2 || got[0].ID != "one" || got[1].ID != "two" {
		t.Fatalf("unexpected declarations: %+v", got)
	}
}

// TestResolveAction verifies commands are derived from current page declarations and choices.
func TestResolveAction(t *testing.T) {
	t.Run("reusable set", func(t *testing.T) {
		source := `{{status id="release" set="workflow"}}`
		action := actionID("release", 3)
		id, value, ok := resolveAction(source, action, func(string, string) (sdk.PluginResourceRecord, error) {
			return sdk.PluginResourceRecord{}, errTestMissing
		})
		if !ok || id != "release" || value != "Done" {
			t.Fatalf("unexpected action resolution: id=%q value=%q ok=%v", id, value, ok)
		}
	})

	t.Run("page local", func(t *testing.T) {
		source := `{{status id="risk" options="Low;High"}}`
		action := actionID("risk", 1)
		id, value, ok := resolveAction(source, action, nil)
		if !ok || id != "risk" || value != "High" {
			t.Fatalf("unexpected local action resolution: id=%q value=%q ok=%v", id, value, ok)
		}
	})
}

// testError provides a dependency-free sentinel error for resource fallbacks.
type testError string

// Error returns the sentinel error text.
func (e testError) Error() string { return string(e) }

const errTestMissing = testError("missing")
