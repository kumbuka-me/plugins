package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestRenderLinksEscapesAndMarksMissingTargets(t *testing.T) {
	output := renderLinks(sdk.PageLinks{
		Backlinks: []sdk.Page{{Slug: "guide/start", Title: `<Guide>`}},
		Outgoing: []sdk.PageLink{
			{TargetSlug: "api/auth", TargetTitle: `Auth & Login`, Exists: true},
			{TargetSlug: `missing page`, Exists: false},
		},
	})

	for _, expected := range []string{`href="/pages/guide/start"`, `&lt;Guide&gt;`, `Auth &amp; Login`, `href="/pages/new?slug=missing+page"`, `class="widget-row broken"`} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}
	if strings.Contains(output, `<Guide>`) {
		t.Fatalf("unescaped title in output: %s", output)
	}
}

func TestRenderLinksEmptyStates(t *testing.T) {
	output := renderLinks(sdk.PageLinks{})
	if !strings.Contains(output, "No pages link here yet.") || !strings.Contains(output, "No wiki links on this page.") {
		t.Fatalf("missing empty state: %s", output)
	}
}
