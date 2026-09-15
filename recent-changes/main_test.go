package main

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderRecentChanges(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderRecent([]sdk.Page{{Slug: "guide/start", Title: "Guide & Start", Tags: []string{"docs"}, UpdatedAt: now.Add(-30 * time.Minute)}}, now, fakeIcon)
	for _, expected := range []string{"Recent changes", `href="/pages/guide/start"`, "Guide &amp; Start", "docs", "30m ago"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}

func TestRenderRecentChangesEmpty(t *testing.T) {
	if output := renderRecent(nil, time.Now(), fakeIcon); !strings.Contains(output, "knowledge base is ready") {
		t.Fatalf("missing empty state: %s", output)
	}
}
