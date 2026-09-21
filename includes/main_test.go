package main

import (
	"errors"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "expand includes: %v", err)
	for _, expected := range []string{"## Restore", "Use {{var:environment}} and {{snippet:warning}}.", "### Detail"} {
		require.Contains(t, got, expected, "expected output to contain %q: %q", expected, got)
	}
	require.NotContains(t, got, "## Verify", "unexpected sibling section in output: %q", got)
}

func TestExpandIncludesRejectsRecursion(t *testing.T) {
	load := pageFixture(map[string]string{"loop": "{{include:loop}}"})

	_, err := expandIncludes("{{include:loop}}", load, nil, 0)
	require.Error(t, err, "expected recursive include error, got %v", err)
	require.Contains(t, err.Error(), "recursive page include", "expected recursive include error, got %v", err)
}

func TestExpandIncludesLeavesFencedSyntaxLiteral(t *testing.T) {
	const source = "```md\n{{include:missing}}\n```"
	got, err := expandIncludes(source, pageFixture(nil), nil, 0)
	require.NoError(t, err, "expand fenced source: %v", err)
	require.Equal(t, source, got, "fenced include changed: got %q want %q", got, source)
}

func TestSectionEndsAtEmptySiblingHeading(t *testing.T) {
	for _, heading := range []string{"#", "#   ", "# ###"} {
		t.Run(heading, func(t *testing.T) {
			section, err := markdownSection("# Guide\nIncluded\n"+heading+"\nOutside", "guide")
			require.NoError(t, err, "section = %q, error = %v", section, err)
			require.Equal(t, "# Guide\nIncluded", section, "section = %q, error = %v", section, err)
		})
	}
}

func TestATXHeadingPreservesLiteralTrailingHashes(t *testing.T) {
	for _, line := range []string{"# C#", "# C# ###"} {
		level, title, ok := atxHeading(line)
		require.True(t, ok, "heading %q = %d, %q, %t", line, level, title, ok)
		require.Equal(t, 1, level, "heading %q = %d, %q, %t", line, level, title, ok)
		require.Equal(t, "C#", title, "heading %q = %d, %q, %t", line, level, title, ok)
	}
	_, _, ok := atxHeading("####### Too many")
	require.False(t, ok, "accepted a seven-level heading")
}
