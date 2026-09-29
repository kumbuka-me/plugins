package main

import (
	"testing"

	"github.com/kumbuka-me/plugins/internal/listindent"
	"github.com/stretchr/testify/require"
)

func TestNumberedListsOnlyMarksOrderedLists(t *testing.T) {
	got, err := listindent.MarkTags(`<ul><li>Bullet</li></ul><ol><li>Number</li></ol>`, "ol", listindent.Class("numbered-list", "compact"))
	require.NoError(t, err)
	require.Contains(t, got, `class="kumbuka-numbered-list-indent-compact"`)
	require.NotContains(t, got, `ul class=`)
}
