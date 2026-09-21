package main

import (
	"testing"

	"github.com/alecthomas/chroma/v2/lexers"
	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformHighlightsKnownLanguage(t *testing.T) {
	result := transform(sdk.RenderRequest{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "go", Source: "package main\n"})
	require.Empty(t, result.Error, "transform returned error: %s", result.Error)
	require.True(t, result.Matched, "unexpected highlight result: %#v", result)
	require.Len(t, result.Parts, 1, "unexpected highlight result: %#v", result)
	require.Contains(t, result.Parts[0].Text, `class="chroma"`, "unexpected highlight result: %#v", result)
}

func TestTransformHighlightsFenceAlias(t *testing.T) {
	result := transform(sdk.RenderRequest{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "sh", Source: "echo Kumbuka\n"})
	require.Empty(t, result.Error, "unexpected alias result: %#v", result)
	require.True(t, result.Matched, "unexpected alias result: %#v", result)
}

func TestFullChromaRegistryIsAvailable(t *testing.T) {
	for _, language := range []string{"go", "typescript", "yaml", "brainfuck"} {
		lexer := lexers.Get(language)
		assert.NotNil(t, lexer, "expected %q lexer to be available", language)
	}
	require.GreaterOrEqual(t, len(lexers.Names(false)), 200, "expected full Chroma lexer registry, got %d lexers", len(lexers.Names(false)))
}

func TestTransformNeverAutoDetectsLanguage(t *testing.T) {
	for _, language := range []string{"", "   ", "not-a-real-language"} {
		result := transform(sdk.RenderRequest{
			APIVersion: sdk.Version,
			Module:     "chroma",
			Stage:      "highlight",
			Language:   language,
			Source:     "package main\n\nfunc main() {}\n",
		})
		require.Empty(t, result.Error, "language %q must stay unmatched instead of being auto-detected: %#v", language, result)
		require.False(t, result.Matched, "language %q must stay unmatched instead of being auto-detected: %#v", language, result)
		require.Len(t, result.Parts, 0, "language %q must stay unmatched instead of being auto-detected: %#v", language, result)
	}
}

func TestTransformUsesLanguagePerCodeBlock(t *testing.T) {
	requests := []sdk.RenderRequest{
		{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "go", Source: "package main\nfunc main() {}\n"},
		{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "python", Source: "def main():\n    pass\n"},
		{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "sql", Source: "SELECT * FROM users;\n"},
	}

	for _, request := range requests {
		result := transform(request)
		require.Empty(t, result.Error, "language %q did not use its explicit lexer: %#v", request.Language, result)
		require.True(t, result.Matched, "language %q did not use its explicit lexer: %#v", request.Language, result)
		require.Len(t, result.Parts, 1, "language %q did not use its explicit lexer: %#v", request.Language, result)
	}
}

func TestTransformLeavesUnknownLanguageUnmatched(t *testing.T) {
	result := transform(sdk.RenderRequest{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "not-a-real-language", Source: "text"})
	require.Empty(t, result.Error, "unexpected unmatched result: %#v", result)
	require.False(t, result.Matched, "unexpected unmatched result: %#v", result)
	require.Len(t, result.Parts, 0, "unexpected unmatched result: %#v", result)
}

func TestTransformRejectsUnsupportedRequest(t *testing.T) {
	for _, request := range []sdk.RenderRequest{
		{APIVersion: sdk.Version, Module: "other", Stage: "highlight", Language: "go", Source: "package main\n"},
		{APIVersion: sdk.Version, Module: "chroma", Stage: "other", Language: "go", Source: "package main\n"},
	} {
		result := transform(request)
		require.NotEmpty(t, result.Error, "unexpected unsupported request result: %#v", result)
		require.False(t, result.Matched, "unexpected unsupported request result: %#v", result)
	}
}

func BenchmarkTransformWarmBash(b *testing.B) {
	request := sdk.RenderRequest{APIVersion: sdk.Version, Module: "chroma", Stage: "highlight", Language: "bash", Source: "set -euo pipefail\necho \"Kumbuka\"\n"}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		result := transform(request)
		if result.Error != "" || !result.Matched {
			b.Fatalf("unexpected result: %#v", result)
		}
	}
}
