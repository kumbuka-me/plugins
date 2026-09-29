package main

import (
	"testing"

	"github.com/kumbuka-me/plugins/internal/listindent"
	"github.com/stretchr/testify/require"
)

func TestBulletedListsOnlyMarksUnorderedLists(t *testing.T) {
	got, err := listindent.MarkTags(`<ul><li>Bullet</li></ul><ol><li>Number</li></ol>`, "ul", listindent.Class("bulleted-list", "wide"))
	require.NoError(t, err)
	require.Contains(t, got, `class="kumbuka-bulleted-list-indent-wide"`)
	require.NotContains(t, got, `ol class=`)
}
