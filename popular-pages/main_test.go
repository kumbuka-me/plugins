package main

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func fakeIcon(name string, _ int) string { return "[" + name + "]" }

func TestRenderPopular(t *testing.T) {
	output := renderPopular([]sdk.Page{{Slug: "guide", Title: "Guide", ViewCount: 42}}, fakeIcon)
	require.Contains(t, output, "Popular pages", "unexpected output: %s", output)
	require.Contains(t, output, "42 views", "unexpected output: %s", output)
}
