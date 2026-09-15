package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestRenderRelated(t *testing.T) {
	output := renderRelated([]sdk.Page{{Slug: "guide/start", Title: `Guide & Start`}})
	for _, expected := range []string{"Related pages", `href="/pages/guide/start"`, `Guide &amp; Start`} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
}

func TestRenderRelatedEmpty(t *testing.T) {
	if output := renderRelated(nil); !strings.Contains(output, "No related pages yet.") {
		t.Fatalf("missing empty state: %s", output)
	}
}
