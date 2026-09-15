package main

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestTypographerOwnsPunctuationTransformation(t *testing.T) {
	got, err := typographHTML(`<p>"Kumbuka" -- documentation... It's useful. '90s.</p>`, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"“Kumbuka”", "– documentation…", "It’s useful.", "’90s"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestTypographerTracksQuotesAcrossInlineMarkupAndSkipsCode(t *testing.T) {
	got, err := typographHTML(`<p>"<em>Kumbuka</em>" '<strong>plugin</strong>' <code>"-- ..."</code></p>`, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`“<em>Kumbuka</em>”`, `‘<strong>plugin</strong>’`} {
		if !strings.Contains(got, want) {
			t.Fatalf("inline quote pair %q was not preserved: %s", want, got)
		}
	}
	if !strings.Contains(got, `<code>"-- ..."</code>`) {
		t.Fatalf("code was transformed: %s", got)
	}
}

func TestTypographerHonorsProgrammingOperatorPolicy(t *testing.T) {
	features := map[string]bool{sdk.RenderPolicyFeature(preserveOperatorsPolicy): true}
	result := transform(sdk.RenderRequest{
		Module:   "typographer",
		Stage:    "postprocess",
		Source:   `<p>"quoted" --> -- --- << >> ...</p>`,
		Features: features,
	})
	if result.Error != "" {
		t.Fatal(result.Error)
	}
	got := result.Parts[0].Text
	for _, want := range []string{"“quoted”", "--&gt; -- --- &lt;&lt; &gt;&gt; …"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}
