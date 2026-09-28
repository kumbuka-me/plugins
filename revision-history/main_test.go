package main

import (
	"testing"
	"time"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestRenderRevision(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{Count: 3, Revisions: []sdk.Revision{{
		Number: 3, Author: `<Alice>`, Message: `Changed & fixed`, CreatedAt: time.Date(2026, 9, 15, 14, 30, 0, 0, time.UTC), AddedLines: 4, RemovedLines: 2,
	}}}, localize.For("en"))
	for _, expected := range []string{"Revision history", `&lt;Alice&gt;`, `Changed &amp; fixed`, "r3", "+4", "−2"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderRevisionEmpty(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{}, localize.For("en"))
	require.Contains(t, output, "No revision history available.", "missing empty state: %s", output)
}

func TestRenderRevisionGerman(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{Revisions: []sdk.Revision{{Number: 1, Author: "Alice"}}}, localize.For("de"))
	require.Contains(t, output, "Versionsverlauf")
	require.Contains(t, output, "Seite erstellt")
}
