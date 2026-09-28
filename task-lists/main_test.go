package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestPresentTaskListsOwnsCheckboxPresentation(t *testing.T) {
	source := `<ul><li><input disabled="" type="checkbox"/> pending</li><li><input checked="" disabled="" type="checkbox"/> done</li></ul>`
	got, err := presentTaskLists(source)
	require.NoError(t, err)

	for _, want := range []string{`class="task-list-item"`, `class="task-list-checkbox"`, `aria-checked="false"`, `>☐</span>`, `class="task-list-checkbox checked"`, `aria-checked="true"`, `>☑</span>`} {
		require.Contains(t, got, want, "missing %q in %s", want, got)
	}
	require.NotContains(t, got, "<input", "checkbox input escaped plugin presentation: %s", got)
}

func TestTaskListPresentationDoesNotRewriteUnrelatedInputs(t *testing.T) {
	source := `<p><input disabled="" type="checkbox"/> ordinary HTML</p>`
	got, err := presentTaskLists(source)
	require.NoError(t, err)
	require.Contains(t, got, "<input", "unrelated input was rewritten: %s", got)
}

func TestTaskListPresentationKeepsLargeUnrelatedHTMLOutsideListItems(t *testing.T) {
	t.Parallel()

	prefix := strings.Repeat(`<p>before content</p>`, 20_000)
	suffix := strings.Repeat(`<p>after content</p>`, 20_000)
	source := prefix + `<ul><li><input checked="" disabled="" type="checkbox"/> done</li></ul>` + suffix

	got, err := presentTaskLists(source)

	require.NoError(t, err)
	require.True(t, strings.HasPrefix(got, prefix), "large prefix changed")
	require.True(t, strings.HasSuffix(got, suffix), "large suffix changed")
	require.Contains(t, got, `class="task-list-checkbox checked"`)
	require.NotContains(t, got[len(prefix):len(got)-len(suffix)], "<input")
}

func TestNestedTaskListsTransformOnce(t *testing.T) {
	t.Parallel()

	source := `<ul><li><input disabled="" type="checkbox"/> parent<ul><li><input checked="" disabled="" type="checkbox"/> child</li></ul></li></ul>`
	got, err := presentTaskLists(source)

	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(got, `class="task-list-item"`))
	require.Equal(t, 2, strings.Count(got, `role="checkbox"`))
	require.NotContains(t, got, "<input")
}

func TestTaskListTransformRejectsWrongStage(t *testing.T) {
	result := transform(sdk.RenderRequest{Module: "presentation", Stage: "preprocess"})
	require.NotEmpty(t, result.Error, "expected unsupported-stage error")
}
