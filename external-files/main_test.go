package main

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

// TestSelectionAndAnnotations verifies configured sources are fetched generically and output is escaped.
func TestSelectionAndAnnotations(t *testing.T) {
	value, ok := parse(`{{external-file source="engineering" path="src/main.go" lines="20-21" note="21:Avoid <script> injection" note="21:Second note"}}`)
	if !ok || value.Invalid {
		t.Fatalf("parse: %+v", value)
	}
	resources := func(resource, key string) (sdk.PluginResourceRecord, error) {
		if resource != "sources" || key != "engineering" {
			t.Fatalf("resource %q %q", resource, key)
		}
		return sdk.PluginResourceRecord{Key: key, Values: map[string]string{
			"provider": "github", "endpoint": "https://api.github.com", "repository": "kumbuka-me/kumbuka", "ref": "main",
			"enabled": "true", "insecure_skip_verify": "false", "token": "secret",
		}}, nil
	}
	httpDo := func(request sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		if request.Method != http.MethodGet || request.URL != "https://api.github.com/repos/kumbuka-me/kumbuka/contents/src/main.go?ref=main" {
			t.Fatalf("request: %+v", request)
		}
		if request.Headers["Authorization"] != "Bearer secret" || request.InsecureSkipVerify || len(request.AllowedPrivateIPs) != 0 {
			t.Fatalf("network policy: %+v", request)
		}
		content := "" + strings.Repeat("before\n", 19) + "<script>alert(1)</script>\n{{include:private}}\nafter"
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		return sdk.HTTPResponse{StatusCode: http.StatusOK, Body: []byte(`{"type":"file","encoding":"base64","size":` + stringInt(len(content)) + `,"content":"` + encoded + `"}`)}, nil
	}
	result := render(value, resources, nil, httpDo)
	out := result.Parts[0].Text
	for _, want := range []string{"&lt;script&gt;", "Line 21:", `class="external-file-marker" title="Annotation 1">1</span>`, `class="external-file-marker" title="Annotation 2">2</span>`, "{{include:private}}", "GitHub", "kumbuka-me/kumbuka", "main"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(out, "<script>") {
		t.Fatal("raw HTML escaped boundary")
	}
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
	for _, args := range []string{`source="a"`, `source="a" path="b" lines="0"`, `source="a" path="b" lines="4-2"`, `source="a" path="b" url="https://x"`, `source="a" source="b" path="x"`, `source="a" path="b" note="0:no"`, `source="a" path="b" note="2:"`, `source="a"path="b"`, `source="a" path="b" lines="999999999999999999999"`} {
		t.Run(args, func(t *testing.T) {
			value, matched := parse("{{external-file " + args + "}}")
			if !matched || !value.Invalid {
				t.Fatalf("accepted %+v", value)
			}
			result := render(value, func(string, string) (sdk.PluginResourceRecord, error) {
				t.Fatal("invalid syntax read resources")
				return sdk.PluginResourceRecord{}, nil
			}, func(string) (sdk.StoredValue, error) {
				t.Fatal("invalid syntax read settings")
				return sdk.StoredValue{}, nil
			}, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
				t.Fatal("invalid syntax fetched")
				return sdk.HTTPResponse{}, nil
			})
			if !strings.Contains(result.Parts[0].Text, "Invalid external file") {
				t.Fatal(result.Parts[0].Text)
			}
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
		if !ok || value.Invalid || value.Start != test.start || value.End != test.end {
			t.Fatalf("selection %+v", value)
		}
	}
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
	if err != nil {
		t.Fatal(err)
	}
	_, err = fetchFile(value, "runbooks/a.md", func(request sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		if request.URL != "https://git.internal.example/api/v4/projects/platform%2Fdocs/repository/files/runbooks%2Fa.md/raw?ref=release&lfs=false" {
			t.Fatalf("url %s", request.URL)
		}
		if request.Headers["PRIVATE-TOKEN"] != "read-token" || !request.InsecureSkipVerify || strings.Join(request.AllowedPrivateIPs, ",") != "10.0.0.12,fd00::12" {
			t.Fatalf("request %+v", request)
		}
		return sdk.HTTPResponse{StatusCode: http.StatusOK, Body: []byte("hello")}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestErrorsAndOutOfRangeNotes verifies provider errors stay opaque and annotation bounds are enforced.
func TestErrorsAndOutOfRangeNotes(t *testing.T) {
	value := options{Source: "error-source", Path: "b", Notes: []annotation{{Line: 4, Text: "note"}}}
	resources := func(string, string) (sdk.PluginResourceRecord, error) {
		return sdk.PluginResourceRecord{}, errors.New("secret token")
	}
	out := render(value, resources, nil, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		return sdk.HTTPResponse{}, errors.New("secret upstream")
	}).Parts[0].Text
	if strings.Contains(out, "secret token") || strings.Contains(out, "secret upstream") || !strings.Contains(out, "unavailable") {
		t.Fatal(out)
	}

	selected, err := selectLines("one\ntwo", 1, 1)
	if err != nil || selected.Start != 1 || selected.Content != "one" {
		t.Fatalf("selection %+v %v", selected, err)
	}
	if _, err := selectLines("one", 4, 4); err == nil {
		t.Fatal("out-of-range line accepted")
	}
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
	if appearance.ReferencePosition != "left" || appearance.ReferenceColor != "purple" || appearance.HighlightReferences || appearance.ShowLineNumbers || appearance.ShowProvider || appearance.ShowBranch {
		t.Fatalf("unexpected settings: %+v", appearance)
	}

	value, ok := parse(`{{external-file source="docs" path="README.md" reference-position="right" reference-color="yellow" highlight-references="true" line-numbers="true" show-provider="true" show-branch="true"}}`)
	if !ok || value.Invalid {
		t.Fatalf("parse: %+v", value)
	}
	appearance = applyPresentationOverrides(appearance, value)
	if appearance.ReferencePosition != "right" || appearance.ReferenceColor != "yellow" || !appearance.HighlightReferences || !appearance.ShowLineNumbers || !appearance.ShowProvider || !appearance.ShowBranch {
		t.Fatalf("unexpected overrides: %+v", appearance)
	}
}

// TestPresentationMarkupUsesSeparateReferenceGutter verifies annotations are not inserted before the source text.
func TestPresentationMarkupUsesSeparateReferenceGutter(t *testing.T) {
	value := options{Path: "README.md", Notes: []annotation{{Line: 2, Text: "Explain this line."}}}
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
		if !strings.Contains(output, expected) {
			t.Fatalf("output does not contain %q: %s", expected, output)
		}
	}

	left := appearance
	left.ReferencePosition = "left"
	left.ShowLineNumbers = false
	left.ShowProvider = false
	left.ShowBranch = false
	output = renderExternalFile(value, source, file, left)
	if !strings.Contains(output, `class="external-file-gutter"><span class="external-file-marker"`) || !strings.Contains(output, `</span><span class="external-file-source">second</span>`) {
		t.Fatalf("left gutter not rendered before source: %s", output)
	}
	for _, absent := range []string{"external-file-number", ">GitLab<", `class="external-file-ref"`} {
		if strings.Contains(output, absent) {
			t.Fatalf("output unexpectedly contains %q: %s", absent, output)
		}
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
		if !matched || !value.Invalid {
			t.Fatalf("accepted presentation override %q: %+v", args, value)
		}
	}
}
