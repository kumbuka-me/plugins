package codepresentation

import (
	"strings"
	"testing"
)

func TestParseRangeAndNotes(t *testing.T) {
	for _, test := range []struct {
		value string
		valid bool
		start int
		end   int
	}{
		{"1", true, 1, 1}, {"3-7", true, 3, 7}, {"0", false, 0, 0},
		{"4-3", false, 0, 0}, {"1-10001", false, 0, 0}, {"abc", false, 0, 0},
		{"2-3-4", false, 0, 0}, {"", false, 0, 0},
	} {
		selected, ok := ParseRange(test.value, 10000)
		if ok != test.valid || selected.Start != test.start || selected.End != test.end {
			t.Errorf("ParseRange(%q): %+v %v", test.value, selected, ok)
		}
	}
	note, ok := ParseNote("2-4:Use <script>alert(1)</script>", 100, 2048)
	if !ok || note.Start != 2 || note.End != 4 {
		t.Fatalf("unexpected parsed note: %+v %v", note, ok)
	}
	if _, ok := ParseNote("2:", 100, 2048); ok {
		t.Fatal("accepted note without description")
	}
	if _, ok := ParseNote("102:outside", 100, 2048); ok {
		t.Fatal("accepted note outside maximum")
	}
}

func TestHighlightsAndBounds(t *testing.T) {
	ranges, ok := ParseHighlights("1,3-5, 7", 100, 3)
	if !ok || len(ranges) != 3 || ranges[1] != (Range{3, 5}) {
		t.Fatalf("unexpected highlights: %+v %v", ranges, ok)
	}
	for _, bad := range []string{"", "0", "4-2", "1,,3", "4-5,9-120", "1,2,3,4"} {
		if _, ok := ParseHighlights(bad, 100, 3); ok {
			t.Errorf("accepted invalid highlights %q", bad)
		}
	}
	if !RangesWithin(ranges, 1, 7) || RangesWithin(ranges, 2, 7) {
		t.Fatal("range validation rejected valid or accepted invalid source interval")
	}
	if !NotesWithin([]Note{{Start: 5, End: 6, Text: "a"}}, 5, 2, 2048) {
		t.Fatal("notes should support original line numbering")
	}
	if NotesWithin([]Note{{Start: 4, End: 6, Text: "a"}}, 5, 2, 2048) {
		t.Fatal("notes must be within source selection")
	}
}

func TestRenderingEscapesAndPreservesSourceText(t *testing.T) {
	block := Block{
		ClassPrefix: "external-file",
		Source:      "before\n<script>bad</script>\nafter",
		StartLine:   20,
		Notes:       []Note{{Start: 21, End: 21, Text: "<img src=x onerror=alert(1)>"}},
		Appearance:  Appearance{ReferencePosition: "right", ReferenceColor: "accent", HighlightReferences: true, ShowLineNumbers: true},
		Labels:      Labels{Line: "Line", Lines: "Lines", Annotation: func(i int) string { return "Annotation " + string(rune('0'+i)) }},
	}
	var output strings.Builder
	WriteCode(&output, block)
	WriteNotes(&output, block)
	text := output.String()
	for _, desired := range []string{`external-file-line-annotated`, `class="external-file-source">&lt;script&gt;bad&lt;/script&gt;`, `title="Annotation 1"`, `&lt;img src=x onerror=alert(1)&gt;`, "Line 21:"} {
		if !strings.Contains(text, desired) {
			t.Errorf("missing %q from generated HTML", desired)
		}
	}
	if strings.Contains(text, "<script>") || strings.Contains(text, "<img") {
		t.Fatal("unescaped HTML in code or note")
	}
}

func TestIndependentHighlightsAndLeftMarkers(t *testing.T) {
	block := Block{
		ClassPrefix: "code-annotation", Source: "one\ntwo", StartLine: 1,
		Highlights: []Range{{Start: 2, End: 2}},
		Notes:      []Note{{Start: 1, End: 1, Text: "note"}},
		Appearance: Appearance{ReferencePosition: "left", HighlightReferences: false},
		Labels:     Labels{Line: "Line", Lines: "Lines"},
	}
	var html strings.Builder
	WriteCode(&html, block)
	text := html.String()
	if strings.Count(text, "code-annotation-line-annotated") != 1 {
		t.Errorf("expected only explicit highlighted line, got %q", text)
	}
	if !strings.Contains(text, `class="code-annotation-gutter"><span class="code-annotation-marker"`) {
		t.Errorf("left note gutter missing: %q", text)
	}
}
