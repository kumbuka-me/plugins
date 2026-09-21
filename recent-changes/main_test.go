package main

import (
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderRecentChanges(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderRecent([]sdk.Page{{Slug: "guide/start", Title: "Guide & Start", Tags: []string{"docs"}, UpdatedAt: now.Add(-30 * time.Minute)}}, now, fakeIcon)
	for _, expected := range []string{"Recent changes", `href="/pages/guide/start"`, "Guide &amp; Start", "docs", "30m ago"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderRecentChangesEmpty(t *testing.T) {
	output := renderRecent(nil, time.Now(), fakeIcon)
	require.Contains(t, output, "knowledge base is ready", "missing empty state: %s", output)
}
