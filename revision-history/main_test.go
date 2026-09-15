package main

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

func TestRenderRevision(t *testing.T) {
	output := renderRevision(sdk.RevisionHistory{Count: 3, Revisions: []sdk.Revision{{
		Number: 3, Author: `<Alice>`, Message: `Changed & fixed`, CreatedAt: time.Date(2026, 9, 15, 14, 30, 0, 0, time.UTC), AddedLines: 4, RemovedLines: 2,
	}}})
	for _, expected := range []string{"Revision history", `&lt;Alice&gt;`, `Changed &amp; fixed`, "r3", "+4", "−2"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}

func TestRenderRevisionEmpty(t *testing.T) {
	if output := renderRevision(sdk.RevisionHistory{}); !strings.Contains(output, "No revision history available.") {
		t.Fatalf("missing empty state: %s", output)
	}
}
