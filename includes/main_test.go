package main

import (
	"errors"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func pageFixture(pages map[string]string) pageLoader {
	return func(slug string) (sdk.PageContent, error) {
		markdown, ok := pages[slug]
		if !ok {
			return sdk.PageContent{}, errors.New("page unavailable")
		}
		return sdk.PageContent{Slug: slug, Markdown: markdown}, nil
	}
}

func TestExpandIncludesNestedSections(t *testing.T) {
	load := pageFixture(map[string]string{
		"runbook": "# Runbook\n\n## Restore\n\n{{include:shared}}\n\n### Detail\n\nMore\n\n## Verify\n\nDone",
		"shared":  "Use {{var:environment}} and {{snippet:warning}}.",
	})

	got, err := expandIncludes("{{include:runbook#Restore}}", load, nil, 0)
	if err != nil {
		t.Fatalf("expand includes: %v", err)
	}
	for _, expected := range []string{"## Restore", "Use {{var:environment}} and {{snippet:warning}}.", "### Detail"} {
		if !strings.Contains(got, expected) {
			t.Fatalf("expected output to contain %q: %q", expected, got)
		}
	}
	if strings.Contains(got, "## Verify") {
		t.Fatalf("unexpected sibling section in output: %q", got)
	}
}

func TestExpandIncludesRejectsRecursion(t *testing.T) {
	load := pageFixture(map[string]string{"loop": "{{include:loop}}"})

	_, err := expandIncludes("{{include:loop}}", load, nil, 0)
	if err == nil || !strings.Contains(err.Error(), "recursive page include") {
		t.Fatalf("expected recursive include error, got %v", err)
	}
}

func TestExpandIncludesLeavesFencedSyntaxLiteral(t *testing.T) {
	const source = "```md\n{{include:missing}}\n```"
	got, err := expandIncludes(source, pageFixture(nil), nil, 0)
	if err != nil {
		t.Fatalf("expand fenced source: %v", err)
	}
	if got != source {
		t.Fatalf("fenced include changed: got %q want %q", got, source)
	}
}
