package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestPlainFencesAreUntouched(t *testing.T) {
	original := "# Title\n\n```go\nfunc main() {}\n```\n\n~~~bash\necho '{code highlight=\"1\"}'\n~~~\n"
	got := annotateFences(original)
	if got != original {
		t.Fatalf("unannotated fences changed:\n%q\nexpected:\n%q", got, original)
	}
}

func TestAnnotatedFenceRendersNotesAndHighlights(t *testing.T) {
	source := "Before\n\n```go\nfunc main() {\n    x := \"<script>\"\n}\n```\n{code highlight=\"1,3\" note=\"2:Load <config>\" note=\"2-3:More details\" reference-position=\"left\" reference-color=\"yellow\" line-numbers=\"false\"}\n\nAfter\n"
	got := annotateFences(source)
	for _, expected := range []string{
		"Before", "After", "code-annotation-reference-left", "code-annotation-color-yellow", "code-annotation-hide-line-numbers",
		`class="code-annotation-marker" title="Annotation 1"`, "&lt;script&gt;", "Load &lt;config&gt;", "Lines 2–3", "code-annotation-line-annotated",
	} {
		if !strings.Contains(got, expected) {
			t.Errorf("missing %q from %q", expected, got)
		}
	}
	if strings.Contains(got, "<script>") || strings.Contains(got, "```go") {
		t.Fatal("source code not escaped or fence not replaced")
	}
	if strings.Count(got, `class="code-annotation-line code-annotation-line-annotated"`) != 3 {
		t.Fatalf("expected 3 shaded lines, got %s", got)
	}
}

func TestMalformedRangesBecomeSafeErrors(t *testing.T) {
	for _, directive := range []string{
		`{code highlight="3"}`, `{code note="4:wrong"}`, `{code note="0:invalid"}`,
		`{code note="1:<img>" reference-color="url()"}`, `{code line-numbers="maybe"}`,
		`{code highlight="1" highlight="2"}`, `{code unknown="bad"}`,
	} {
		source := "```go\none\ntwo\n```\n" + directive + "\n"
		got := annotateFences(source)
		if !strings.Contains(got, `class="code-annotation-error"`) || strings.Contains(got, "<img>") {
			t.Errorf("invalid directive %q was not safely rejected: %s", directive, got)
		}
	}
}

func TestEscapedQuotedNotes(t *testing.T) {
	value, err := parseDirective(`{code note="1:A \"quoted\" description" highlight="1"}`)
	if err != nil || len(value.Notes) != 1 || value.Notes[0].Text != `A "quoted" description` {
		t.Fatalf("could not decode quoted note: %+v, %v", value, err)
	}
}

func TestClosingFenceMustMatchAndDirectiveMustFollowImmediately(t *testing.T) {
	for _, input := range []string{
		"````markdown\n```go\na\n```\n{code highlight=\"1\"}\n````\n",
		"```go\na\n```\n\n{code highlight=\"1\"}\n",
		"~~~go\na\n```\n{code highlight=\"1\"}\n",
	} {
		if got := annotateFences(input); got != input {
			t.Errorf("changed unannotated input: %q", got)
		}
	}
}

func TestTildeFencesAndTransforms(t *testing.T) {
	source := "~~~bash\necho hi\n~~~\n{code highlight=\"1\"}\n"
	got := transform(sdk.RenderRequest{Module: "code-annotations", Stage: "preprocess", Source: source})
	if got.Error != "" || len(got.Parts) != 1 || !strings.Contains(got.Parts[0].Text, `class="code-annotation code-annotation-reference-right`) {
		t.Fatalf("unexpected annotated tilde fence: %+v", got)
	}
	if transform(sdk.RenderRequest{Module: "code-annotations", Stage: "postprocess", Source: source}).Error == "" {
		t.Fatal("accepted unsupported processing stage")
	}
}

func TestVisualMacroAndInvalidInput(t *testing.T) {
	source := `{{annotated-code language="go" source="func main() {\n    println(\"hello\")\n}" highlight="2" note="2:Print hello." reference-position="left"}}`
	value, ok := parseMacro(source)
	if !ok || value.Invalid || value.Language != "go" || !strings.Contains(value.Source, "println") || len(value.Notes) != 1 {
		t.Fatalf("unexpected visual macro parsing: %+v %t", value, ok)
	}
	output, err := renderBlock(value.Language, value.Source, value)
	if err != nil || !strings.Contains(output, "code-annotation-reference-left") || !strings.Contains(output, "Print hello.") {
		t.Fatalf("unexpected visual macro output: %s %v", output, err)
	}
	malformed, matched := parseMacro(`{{annotated-code language="go" source="hi" highlight="100"}}`)
	if !matched || malformed.Invalid {
		t.Fatalf("malformed line reference should be validated during rendering: %+v %v", malformed, matched)
	}
	if _, err := renderBlock(malformed.Language, malformed.Source, malformed); err == nil {
		t.Fatal("accepted out-of-range visual macro highlight")
	}
	malformed, matched = parseMacro(`{{annotated-code language="go" source="bad" unknown="bad"}}`)
	if !matched || !malformed.Invalid {
		t.Fatal("accepted unsupported visual macro attribute")
	}
	if _, matched = parseMacro(`{{external-file source="a"}}`); matched {
		t.Fatal("claimed unrelated macro")
	}
}
