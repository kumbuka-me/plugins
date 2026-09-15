package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestPresentTaskListsOwnsCheckboxPresentation(t *testing.T) {
	source := `<ul><li><input disabled="" type="checkbox"/> pending</li><li><input checked="" disabled="" type="checkbox"/> done</li></ul>`
	got, err := presentTaskLists(source)
	if err != nil {
		t.Fatal(err)
	}

	for _, want := range []string{`class="task-list-item"`, `class="task-list-checkbox"`, `aria-checked="false"`, `>☐</span>`, `class="task-list-checkbox checked"`, `aria-checked="true"`, `>☑</span>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
	if strings.Contains(got, "<input") {
		t.Fatalf("checkbox input escaped plugin presentation: %s", got)
	}
}

func TestTaskListPresentationDoesNotRewriteUnrelatedInputs(t *testing.T) {
	source := `<p><input disabled="" type="checkbox"/> ordinary HTML</p>`
	got, err := presentTaskLists(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "<input") {
		t.Fatalf("unrelated input was rewritten: %s", got)
	}
}

func TestTaskListTransformRejectsWrongStage(t *testing.T) {
	result := transform(sdk.RenderRequest{Module: "presentation", Stage: "preprocess"})
	if result.Error == "" {
		t.Fatal("expected unsupported-stage error")
	}
}
