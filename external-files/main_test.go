package main

import (
	"errors"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestSelectionAndAnnotations(t *testing.T) {
	o, ok := parse(`{{external-file source="engineering" path="src/main.go" lines="20-21" note="21:Avoid <script> injection" note="21:Second note"}}`)
	if !ok || o.Invalid {
		t.Fatalf("parse: %+v", o)
	}
	result := render(o, func(q sdk.ExternalFileRequest) (sdk.ExternalFile, error) {
		if q.Source != "engineering" || q.Path != "src/main.go" || q.Start != 20 || q.End != 21 {
			t.Fatalf("request %+v", q)
		}
		return sdk.ExternalFile{Start: 20, Content: "<script>alert(1)</script>\n{{include:private}}"}, nil
	})
	out := result.Parts[0].Text
	for _, want := range []string{"&lt;script&gt;", "Line 21:", "[1]", "[2]", "{{include:private}}"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "<script>") {
		t.Fatal("raw HTML escaped boundary")
	}
}

func TestInvalidSyntaxNeverFetches(t *testing.T) {
	for _, args := range []string{`source="a"`, `source="a" path="b" lines="0"`, `source="a" path="b" lines="4-2"`, `source="a" path="b" url="https://x"`, `source="a" source="b" path="x"`, `source="a" path="b" note="0:no"`, `source="a" path="b" note="2:"`, `source="a"path="b"`, `source="a" path="b" lines="999999999999999999999"`} {
		t.Run(args, func(t *testing.T) {
			o, matched := parse("{{external-file " + args + "}}")
			if !matched || !o.Invalid {
				t.Fatalf("accepted %+v", o)
			}
			render(o, func(sdk.ExternalFileRequest) (sdk.ExternalFile, error) {
				t.Fatal("invalid syntax fetched")
				return sdk.ExternalFile{}, nil
			})
		})
	}
}
func TestWholeAndSingleLine(t *testing.T) {
	for _, test := range []struct {
		suffix     string
		start, end int
	}{{"", 0, 0}, {` lines="3"`, 3, 3}} {
		o, ok := parse(`{{external-file source="docs" path="README.md"` + test.suffix + `}}`)
		if !ok || o.Invalid || o.Start != test.start || o.End != test.end {
			t.Fatalf("selection %+v", o)
		}
	}
}
func TestErrorsAndOutOfRangeNotes(t *testing.T) {
	o := options{Source: "a", Path: "b", Notes: []annotation{{Line: 4, Text: "note"}}}
	out := render(o, func(sdk.ExternalFileRequest) (sdk.ExternalFile, error) {
		return sdk.ExternalFile{Start: 1, Content: "one"}, nil
	}).Parts[0].Text
	if !strings.Contains(out, "outside") {
		t.Fatal(out)
	}
	out = render(o, func(sdk.ExternalFileRequest) (sdk.ExternalFile, error) {
		return sdk.ExternalFile{}, errors.New("secret token")
	}).Parts[0].Text
	if strings.Contains(out, "secret token") || !strings.Contains(out, "unavailable") {
		t.Fatal(out)
	}
}
