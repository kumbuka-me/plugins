package main

import (
	"os"
	"testing"
	"time"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

// TestRenderRevisionKeepsOnlyCompactSummary verifies content revisions hide redundant descriptions.
func TestRenderRevisionKeepsOnlyCompactSummary(t *testing.T) {
	t.Parallel()

	output := renderRevision(sdk.RevisionHistory{Count: 3, Revisions: []sdk.Revision{{
		Number: 3, Author: `<Alice>`, Message: `Markdown changed & more`,
		CreatedAt: time.Date(2026, 11, 4, 16, 9, 0, 0, time.UTC), AddedLines: 23, RemovedLines: 19,
	}}}, localize.For("en"))

	for _, expected := range []string{
		"Latest revision", `class="widget-entry--clickable"`, `class="prose"`, `class="kumbuka-revision-card"`, `&lt;Alice&gt;`,
		"r3", "2026-11-04 · 16:09", "+23", "−19", "lines",
	} {
		require.Contains(t, output, expected)
	}
	require.NotContains(t, output, `class="widget-message"`)
	require.NotContains(t, output, "Markdown changed")
	require.NotContains(t, output, "Markdown changed &amp; more")
	require.NotContains(t, output, "widget-revision-")
}

// TestRenderRevisionEmpty verifies the summary is not clickable without revision history.
func TestRenderRevisionEmpty(t *testing.T) {
	t.Parallel()

	output := renderRevision(sdk.RevisionHistory{}, localize.For("en"))
	require.Contains(t, output, "No revision history available.")
	require.NotContains(t, output, "widget-entry--clickable")
}

// TestRenderRevisionGerman verifies localized dates and the page-creation label.
func TestRenderRevisionGerman(t *testing.T) {
	t.Parallel()

	output := renderRevision(sdk.RevisionHistory{Revisions: []sdk.Revision{{
		Number: 1, Author: "Alice", CreatedAt: time.Date(2026, 11, 4, 16, 9, 0, 0, time.UTC),
	}}}, localize.For("de"))
	require.Contains(t, output, "Letzte Version")
	require.Contains(t, output, "04.11.2026 · 16:09")
	require.Contains(t, output, "Seite erstellt")
	require.NotContains(t, output, "+0")
	require.NotContains(t, output, "−0")
}

// TestRenderRevisionMetadataOnlyLabelsChangedFields verifies metadata summaries stay concise and escaped.
func TestRenderRevisionMetadataOnlyLabelsChangedFields(t *testing.T) {
	t.Parallel()

	revisions := []struct {
		message string
		want    string
	}{
		{message: "Changed: title, tags", want: "Geändert: title, tags"},
		{message: "Revision restored · Changed: status", want: "Geändert: status"},
		{message: "Changed: &lt;unsafe&gt;", want: "Geändert: &amp;lt;unsafe&amp;gt;"},
		{message: "Manually edited", want: "Seitendetails geändert"},
	}
	for _, test := range revisions {
		t.Run(test.message, func(t *testing.T) {
			t.Parallel()
			output := renderRevision(sdk.RevisionHistory{Revisions: []sdk.Revision{{
				Number: 4, Author: "admin", Message: test.message,
			}}}, localize.For("de"))
			require.Contains(t, output, test.want)
			require.NotContains(t, output, "Keine Markdown-Änderungen")
		})
	}
}

// TestRevisionResultOpensExactRevision verifies the host owns the entire clickable dialog action.
func TestRevisionResultOpensExactRevision(t *testing.T) {
	t.Parallel()

	history := sdk.RevisionHistory{Revisions: []sdk.Revision{{Number: 3, Author: "Alice"}}}
	result := revisionResult(history, "guide/notes", localize.For("en"))

	require.Len(t, result.Actions, 1)
	require.Equal(t, "dialog", result.Actions[0].Kind)
	require.Equal(t, "View changes", result.Actions[0].Label)
	require.Empty(t, result.Actions[0].Icon)
	require.Equal(t, "/revisions/guide/notes?revision=3", result.Actions[0].URL)
}

// TestRevisionResultOmitsEmptyAction verifies no comparison is offered without revisions.
func TestRevisionResultOmitsEmptyAction(t *testing.T) {
	t.Parallel()

	result := revisionResult(sdk.RevisionHistory{}, "guide", localize.For("de"))
	require.Empty(t, result.Actions)
	require.Contains(t, result.Parts[0].Text, "Kein Versionsverlauf verfügbar.")
}

// TestPresentationIsPackagedAsContentStyle ensures the widget owns its layout asset.
func TestPresentationIsPackagedAsContentStyle(t *testing.T) {
	t.Parallel()

	manifest, err := os.ReadFile("plugin.yaml")
	require.NoError(t, err)
	require.Contains(t, string(manifest), "- type: content-style\n    id: presentation\n    css: revision-history.css")

	stylesheet, err := os.ReadFile("assets/revision-history.css")
	require.NoError(t, err)
	require.Contains(t, string(stylesheet), ".prose .kumbuka-revision-card")
	require.Contains(t, string(stylesheet), ".prose .kumbuka-revision-added")
	require.NotContains(t, string(stylesheet), "@import")
	require.NotContains(t, string(stylesheet), "position:")
}
