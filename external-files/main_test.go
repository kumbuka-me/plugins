package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectionAndAnnotations verifies configured sources are fetched generically and output is escaped.
func TestSelectionAndAnnotations(t *testing.T) {
	value, ok := parse(`{{external-file source="engineering" path="src/main.go" lines="20-21" note="21:Avoid <script> injection" note="21:Second note"}}`)
	require.True(t, ok, "parse: %+v", value)
	require.False(t, value.Invalid, "parse: %+v", value)
	resources := func(resource, key string) (sdk.PluginResourceRecord, error) {
		require.Equal(t, "sources", resource, "resource %q %q", resource, key)
		require.Equal(t, "engineering", key, "resource %q %q", resource, key)
		return sdk.PluginResourceRecord{Key: key, Values: map[string]string{
			"provider": "github", "endpoint": "https://api.github.com", "repository": "kumbuka-me/kumbuka", "ref": "main",
			"enabled": "true", "insecure_skip_verify": "false", "token": "secret",
		}}, nil
	}
	httpDo := func(request sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		require.Equal(t, http.MethodGet, request.Method, "request: %+v", request)
		require.Equal(t, "https://api.github.com/repos/kumbuka-me/kumbuka/contents/src/main.go?ref=main", request.URL, "request: %+v", request)
		require.Equal(t, "Bearer secret", request.Headers["Authorization"], "network policy: %+v", request)
		require.False(t, request.InsecureSkipVerify, "network policy: %+v", request)
		require.Len(t, request.AllowedPrivateIPs, 0, "network policy: %+v", request)
		content := "" + strings.Repeat("before\n", 19) + "<script>alert(1)</script>\n{{include:private}}\nafter"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		return sdk.HTTPResponse{StatusCode: http.StatusOK, Body: []byte(`{"type":"file","encoding":"base64","size":` + stringInt(len(content)) + `,"content":"` + encoded + `"}`)}, nil
	}
	result := render(value, resources, nil, httpDo)
	out := result.Parts[0].Text
	for _, want := range []string{"&lt;script&gt;", "Line 21:", `class="external-file-marker" title="Annotation 1">1</span>`, `class="external-file-marker" title="Annotation 2">2</span>`, "{{include:private}}", "GitHub", "kumbuka-me/kumbuka", "main"} {
		assert.Contains(t, out, want, "missing %q", want)
	}
	require.NotContains(t, out, "<script>", "raw HTML escaped boundary")
}

// stringInt formats a small integer for inline JSON fixtures.
func stringInt(value int) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buffer [32]byte
	position := len(buffer)
	for value > 0 {
		position--
		buffer[position] = digits[value%10]
		value /= 10
	}
	return string(buffer[position:])
}

// TestInvalidSyntaxNeverFetches verifies malformed matching macros cannot reach plugin resources or HTTP.
func TestInvalidSyntaxNeverFetches(t *testing.T) {
	for _, args := range []string{`source="a"`, `source="a" path="b" lines="0"`, `source="a" path="b" lines="4-2"`, `source="a" path="b" url="https://x"`, `source="a" source="b" path="x"`, `source="a" path="b" note="0:no"`, `source="a" path="b" note="2:"`, `source="a"path="b"`, `source="a" path="b" lines="999999999999999999999"`, `source="\xff" path="b"`} {
		t.Run(args, func(t *testing.T) {
			value, matched := parse("{{external-file " + args + "}}")
			require.True(t, matched, "accepted %+v", value)
			require.True(t, value.Invalid, "accepted %+v", value)
			result := render(value, func(string, string) (sdk.PluginResourceRecord, error) {
				require.FailNow(t, "invalid syntax read resources")
				return sdk.PluginResourceRecord{}, nil
			}, func(string) (sdk.StoredValue, error) {
				require.FailNow(t, "invalid syntax read settings")
				return sdk.StoredValue{}, nil
			}, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
				require.FailNow(t, "invalid syntax fetched")
				return sdk.HTTPResponse{}, nil
			})
			require.Contains(t, result.Parts[0].Text, "Invalid external file")
		})
	}
}

// TestWholeAndSingleLine verifies macro line selections retain their original bounds.
func TestWholeAndSingleLine(t *testing.T) {
	for _, test := range []struct {
		suffix     string
		start, end int
	}{{"", 0, 0}, {` lines="3"`, 3, 3}} {
		value, ok := parse(`{{external-file source="docs" path="README.md"` + test.suffix + `}}`)
		require.True(t, ok, "selection %+v", value)
		require.False(t, value.Invalid, "selection %+v", value)
		require.Equal(t, test.start, value.Start, "selection %+v", value)
		require.Equal(t, test.end, value.End, "selection %+v", value)
	}
}

// TestAnnotationRanges verifies notes accept inclusive ranges while preserving single-line syntax.
func TestAnnotationRanges(t *testing.T) {
	value, ok := parse(`{{external-file source="docs" path="README.md" lines="10-20" note="12-15:Explain this block." note="18:Explain this line."}}`)
	require.True(t, ok, "parse: %+v", value)
	require.False(t, value.Invalid, "parse: %+v", value)
	require.Equal(t, []annotation{
		{Start: 12, End: 15, Text: "Explain this block."},
		{Start: 18, End: 18, Text: "Explain this line."},
	}, value.Notes)

	file := selectedFile{Start: 10, Content: strings.Join([]string{
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19", "20",
	}, "\n")}
	require.True(t, annotationsWithinSelection(value.Notes, file))
	output := renderExternalFile(value, source{}, file, defaultPresentation())
	require.Equal(t, 5, strings.Count(output, "external-file-line-annotated"), output)
	require.Contains(t, output, "<strong>Lines 12–15:</strong>")
	require.Contains(t, output, "<strong>Line 18:</strong>")

	value.Notes[0].End = 21
	require.False(t, annotationsWithinSelection(value.Notes, file))
}

// TestSourceNetworkSettings verifies provider, private-network, and TLS settings shape generic HTTP requests.
func TestSourceNetworkSettings(t *testing.T) {
	resources := func(string, string) (sdk.PluginResourceRecord, error) {
		return sdk.PluginResourceRecord{Key: "internal", Values: map[string]string{
			"provider": "gitlab", "endpoint": "https://git.internal.example/api/v4", "repository": "platform/docs", "ref": "release",
			"private_ips": "10.0.0.12, fd00::12", "token": "read-token", "insecure_skip_verify": "true", "enabled": "true",
		}}, nil
	}
	value, err := loadSource("internal", resources)
	require.NoError(t, err)
	_, err = fetchFile(value, "runbooks/a.md", func(request sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		require.Equal(t, "https://git.internal.example/api/v4/projects/platform%2Fdocs/repository/files/runbooks%2Fa.md/raw?ref=release&lfs=false", request.URL, "url %s", request.URL)
		require.Equal(t, "read-token", request.Headers["PRIVATE-TOKEN"], "request %+v", request)
		require.True(t, request.InsecureSkipVerify, "request %+v", request)
		require.Equal(t, "10.0.0.12,fd00::12", strings.Join(request.AllowedPrivateIPs, ","), "request %+v", request)
		return sdk.HTTPResponse{StatusCode: http.StatusOK, Body: []byte("hello")}, nil
	})
	require.NoError(t, err)
}

// TestErrorsAndOutOfRangeNotes verifies provider errors stay opaque and annotation bounds are enforced.
func TestErrorsAndOutOfRangeNotes(t *testing.T) {
	value := options{Source: "error-source", Path: "b", Notes: []annotation{{Start: 4, End: 4, Text: "note"}}}
	resources := func(string, string) (sdk.PluginResourceRecord, error) {
		return sdk.PluginResourceRecord{}, errors.New("secret token")
	}
	out := render(value, resources, nil, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		return sdk.HTTPResponse{}, errors.New("secret upstream")
	}).Parts[0].Text
	require.NotContains(t, out, "secret token")
	require.NotContains(t, out, "secret upstream")
	require.Contains(t, out, "unavailable")

	selected, err := selectLines("one\ntwo", 1, 1)
	require.NoError(t, err, "selection %+v %v", selected, err)
	require.Equal(t, 1, selected.Start, "selection %+v %v", selected, err)
	require.Equal(t, "one", selected.Content, "selection %+v %v", selected, err)
	_, err = selectLines("one", 4, 4)
	require.Error(t, err, "out-of-range line accepted")
}

// TestPresentationSettingsAndOverrides verifies plugin defaults and per-embed overrides stay inside External Files.
func TestPresentationSettingsAndOverrides(t *testing.T) {
	settings := func(key string) (sdk.StoredValue, error) {
		values := map[string]string{
			"appearance.reference_position":         "left",
			"appearance.reference_color":            "purple",
			"appearance.highlight_referenced_lines": "false",
			"appearance.show_line_numbers":          "false",
			"appearance.show_provider":              "false",
			"appearance.show_branch":                "false",
		}
		value, ok := values[key]
		return sdk.StoredValue{Found: ok, Value: []byte(value)}, nil
	}

	appearance := loadPresentation(settings)
	require.Equal(t, "left", appearance.ReferencePosition, "unexpected settings: %+v", appearance)
	require.Equal(t, "purple", appearance.ReferenceColor, "unexpected settings: %+v", appearance)
	require.False(t, appearance.HighlightReferences, "unexpected settings: %+v", appearance)
	require.False(t, appearance.ShowLineNumbers, "unexpected settings: %+v", appearance)
	require.False(t, appearance.ShowProvider, "unexpected settings: %+v", appearance)
	require.False(t, appearance.ShowBranch, "unexpected settings: %+v", appearance)

	value, ok := parse(`{{external-file source="docs" path="README.md" reference-position="right" reference-color="yellow" highlight-references="true" line-numbers="true" show-provider="true" show-branch="true"}}`)
	require.True(t, ok, "parse: %+v", value)
	require.False(t, value.Invalid, "parse: %+v", value)
	appearance = applyPresentationOverrides(appearance, value)
	require.Equal(t, "right", appearance.ReferencePosition, "unexpected overrides: %+v", appearance)
	require.Equal(t, "yellow", appearance.ReferenceColor, "unexpected overrides: %+v", appearance)
	require.True(t, appearance.HighlightReferences, "unexpected overrides: %+v", appearance)
	require.True(t, appearance.ShowLineNumbers, "unexpected overrides: %+v", appearance)
	require.True(t, appearance.ShowProvider, "unexpected overrides: %+v", appearance)
	require.True(t, appearance.ShowBranch, "unexpected overrides: %+v", appearance)
}

// TestPresentationMarkupUsesSeparateReferenceGutter verifies annotations are not inserted before the source text.
func TestPresentationMarkupUsesSeparateReferenceGutter(t *testing.T) {
	value := options{Path: "README.md", Notes: []annotation{{Start: 2, End: 2, Text: "Explain this line."}}}
	source := source{Provider: "gitlab", Repository: "platform/docs", Ref: "main"}
	file := selectedFile{Start: 1, Content: "first\nsecond"}
	appearance := defaultPresentation()

	output := renderExternalFile(value, source, file, appearance)
	for _, expected := range []string{
		"external-file-reference-right",
		"external-file-color-accent",
		">GitLab<",
		">platform/docs<",
		">main<",
		"external-file-line-annotated",
		`class="external-file-source">second</span><span class="external-file-gutter">`,
		`class="external-file-marker" title="Annotation 1">1</span>`,
	} {
		require.Contains(t, output, expected, "output does not contain %q: %s", expected, output)
	}

	left := appearance
	left.ReferencePosition = "left"
	left.ShowLineNumbers = false
	left.ShowProvider = false
	left.ShowBranch = false
	output = renderExternalFile(value, source, file, left)
	require.Contains(t, output, `class="external-file-gutter"><span class="external-file-marker"`, "left gutter not rendered before source: %s", output)
	require.Contains(t, output, `</span><span class="external-file-source">second</span>`, "left gutter not rendered before source: %s", output)
	require.Contains(t, output, "external-file-hide-line-numbers")
	for _, absent := range []string{">GitLab<", `class="external-file-ref"`} {
		require.NotContains(t, output, absent, "output unexpectedly contains %q: %s", absent, output)
	}
}

// TestPresentationOverrideValidationRejectsUnknownValues verifies unsafe style values never reach generated class names.
func TestPresentationOverrideValidationRejectsUnknownValues(t *testing.T) {
	for _, args := range []string{
		`source="a" path="b" reference-position="center"`,
		`source="a" path="b" reference-color="url(evil)"`,
		`source="a" path="b" line-numbers="maybe"`,
	} {
		value, matched := parse("{{external-file " + args + "}}")
		require.True(t, matched, "accepted presentation override %q: %+v", args, value)
		require.True(t, value.Invalid, "accepted presentation override %q: %+v", args, value)
	}
}
