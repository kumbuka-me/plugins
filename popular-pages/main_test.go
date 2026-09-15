package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderPopular(t *testing.T) {
	output := renderPopular([]sdk.Page{{Slug: "guide", Title: "Guide", ViewCount: 42}}, fakeIcon)
	if !strings.Contains(output, "Popular pages") || !strings.Contains(output, "42 views") {
		t.Fatalf("unexpected output: %s", output)
	}
}
