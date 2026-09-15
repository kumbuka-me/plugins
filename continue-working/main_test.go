package main

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderContinueWorking(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderContinueWorking(
		[]sdk.PageDraft{{PageID: 1, PageSlug: "guide/start", Title: "Draft & Guide", Stale: true, UpdatedAt: now.Add(-10 * time.Minute)}},
		[]sdk.RecentEdit{{Page: sdk.Page{Slug: "api", Title: "API", UpdatedAt: now.Add(-2 * time.Hour)}, RevisionMessage: "Fix & polish"}},
		now,
		fakeIcon,
	)
	for _, expected := range []string{"Continue working", `href="/edit/guide/start"`, "Draft &amp; Guide", "10m ago", "Page changed since draft started", "Fix &amp; polish", "2h ago"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}
