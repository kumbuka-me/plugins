package main

import (
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestRenderRevision(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{Count: 3, Revisions: []sdk.Revision{{
		Number: 3, Author: `<Alice>`, Message: `Changed & fixed`, CreatedAt: time.Date(2026, 9, 15, 14, 30, 0, 0, time.UTC), AddedLines: 4, RemovedLines: 2,
	}}})
	for _, expected := range []string{"Revision history", `&lt;Alice&gt;`, `Changed &amp; fixed`, "r3", "+4", "−2"} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}
}

func TestRenderRevisionEmpty(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{})
	require.Contains(t, output, "No revision history available.", "missing empty state: %s", output)
}
