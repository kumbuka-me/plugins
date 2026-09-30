package main

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kumbuka-me/plugins/internal/listindent"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestPresentChecklistsOwnsInteractiveCheckboxPresentation(t *testing.T) {
	source := `<ul><li><input disabled="" type="checkbox"/> pending</li><li><input checked="" disabled="" type="checkbox"/> done</li></ul>`
	got, err := presentChecklists(source, listindent.Default)
	require.NoError(t, err)

	for _, want := range []string{`class="kumbuka-checklist-browser"`, `class="checklist-item"`, `checklist-checkbox checklist-action__toggle-0-0`, `aria-checked="false"`, `checklist-checkbox checked checklist-action__toggle-1-1`, `aria-checked="true"`, `>✓</button>`} {
		require.Contains(t, got, want, "missing %q in %s", want, got)
	}
	require.NotContains(t, got, "<input", "checkbox input escaped plugin presentation: %s", got)
}

func TestTaskListPresentationDoesNotRewriteUnrelatedInputs(t *testing.T) {
	source := `<p><input disabled="" type="checkbox"/> ordinary HTML</p>`
	got, err := presentChecklists(source, listindent.Default)
	require.NoError(t, err)
	require.Contains(t, got, "<input", "unrelated input was rewritten: %s", got)
}

func TestTaskListPresentationKeepsLargeUnrelatedHTMLOutsideListItems(t *testing.T) {
	t.Parallel()

	prefix := strings.Repeat(`<p>before content</p>`, 20_000)
	suffix := strings.Repeat(`<p>after content</p>`, 20_000)
	source := prefix + `<ul><li><input checked="" disabled="" type="checkbox"/> done</li></ul>` + suffix

	got, err := presentChecklists(source, listindent.Default)

	require.NoError(t, err)
	require.True(t, strings.HasPrefix(got, prefix), "large prefix changed")
	require.True(t, strings.HasSuffix(got, suffix), "large suffix changed")
	require.Contains(t, got, `class="checklist-checkbox checked checklist-action__`)
	require.NotContains(t, got[len(prefix):len(got)-len(suffix)], "<input")
}

func TestNestedTaskListsTransformOnce(t *testing.T) {
	t.Parallel()

	source := `<ul><li><input disabled="" type="checkbox"/> parent<ul><li><input checked="" disabled="" type="checkbox"/> child</li></ul></li></ul>`
	got, err := presentChecklists(source, listindent.Default)

	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(got, `class="checklist-item"`))
	require.Equal(t, 2, strings.Count(got, `role="checkbox"`))
	require.Equal(t, 1, strings.Count(got, `class="kumbuka-checklist-browser"`))
	require.NotContains(t, got, "<input")
}

func TestToggleTaskMarkerUpdatesOnlyRequestedItem(t *testing.T) {
	source := "- [ ] first\n\n```md\n- [ ] example\n```\n> - [X] second\n"
	got, err := toggleTaskMarker(source, 1, true, checklistFingerprint([]bool{false, true}))
	require.NoError(t, err)
	require.Equal(t, "- [ ] first\n\n```md\n- [ ] example\n```\n> - [ ] second\n", got)
}

func TestToggleTaskMarkerRejectsStaleState(t *testing.T) {
	_, err := toggleTaskMarker("- [x] done\n", 0, false, checklistFingerprint([]bool{true}))
	require.Error(t, err)
}

func TestToggleTaskMarkerRejectsStaleChecklistShape(t *testing.T) {
	_, err := toggleTaskMarker("- [ ] inserted\n- [ ] original\n", 0, false, checklistFingerprint([]bool{false}))
	require.Error(t, err)
}

func TestTaskListTransformRejectsWrongStage(t *testing.T) {
	result := transform(sdk.RenderRequest{Module: "presentation", Stage: "preprocess"})
	require.NotEmpty(t, result.Error, "expected unsupported-stage error")
}

func TestToggleTaskMarkerKeepsFenceLikeCodeInsideFence(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(strconv.Quote(newline), func(t *testing.T) {
			source := strings.Join([]string{"```md", "```not-a-closing-fence", "- [ ] example", "```", "- [ ] real", ""}, newline)
			got, err := toggleTaskMarker(source, 0, false, checklistFingerprint([]bool{false}))
			require.NoError(t, err)
			require.Equal(t, strings.Replace(source, "- [ ] real", "- [x] real", 1), got)
		})
	}
}
