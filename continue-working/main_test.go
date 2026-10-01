package main

import (
	"testing"
	"time"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderContinueWorking(t *testing.T) {
	now := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC)
	output := renderContinueWorking(
		[]sdk.PageDraft{{URL: "/kumbuka/edit/guide/start", PageID: 1, PageSlug: "guide/start", Title: "Draft & Guide", Stale: true, UpdatedAt: now.Add(-10 * time.Minute)}},
		[]sdk.RecentEdit{{Page: sdk.Page{EditURL: "/kumbuka/edit/api", Slug: "api", Title: "API", UpdatedAt: now.Add(-2 * time.Hour)}, RevisionMessage: "Fix & polish"}},
		now,
		fakeIcon,
		localize.For("en"),
	)
	for _, expected := range []string{"Continue working", `href="/kumbuka/edit/guide/start"`, `href="/kumbuka/edit/api"`, "Draft &amp; Guide", "10m ago", "Page changed since draft started", "Fix &amp; polish", "2h ago"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderContinueWorkingGerman(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	output := renderContinueWorking(
		[]sdk.PageDraft{{URL: "/kumbuka/pages/new", Title: "", Stale: true, UpdatedAt: now.Add(-10 * time.Minute)}},
		nil,
		now,
		fakeIcon,
		localize.For("de"),
	)
	for _, expected := range []string{"Weiterarbeiten", "Entwürfe", "Ohne Titel", "Privater Entwurf", "vor 10 Min.", "Seite wurde seit Beginn des Entwurfs geändert", "Letzte Bearbeitungen"} {
		require.Contains(t, output, expected)
	}
}
