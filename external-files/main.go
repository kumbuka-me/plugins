// Package main implements external repository file embeds using generic Kumbuka plugin capabilities.
package main

import (
	"strconv"
	"strings"

	"github.com/kumbuka-me/plugins/internal/localize"
	"github.com/kumbuka-me/plugins/internal/macroargs"

	sdk "github.com/kumbuka-me/sdk"
)

// main is the WASI package entry point.
func main() {}

// init registers the external-file macro and cache administration action with the Kumbuka SDK.
func init() {
	sdk.RegisterLocalizedMacro("external-files", parse, func(invocation sdk.LocalizedMacro[options]) (sdk.Result, error) {
		return render(invocation.Value, sdk.Resources().Get, sdk.Settings().Get, sdk.HTTP().Do, localize.For(invocation.Locale)), nil
	})
	sdk.RegisterAdminAction("refresh-cache", func() error {
		refreshCache()
		return nil
	})
}

// annotation describes one numbered note attached to an inclusive original source range.
type annotation struct {
	// Start is the first one-based original source line receiving the note.
	Start int
	// End is the final one-based original source line receiving the note.
	End int
	// Text is the plain-text annotation shown below the source block.
	Text string
}

// optionalBool preserves whether a per-embed boolean override was supplied.
type optionalBool struct {
	// Set reports whether the macro explicitly supplied an override.
	Set bool
	// Value is the requested boolean when Set is true.
	Value bool
}

// options contains one parsed external-file macro invocation.
type options struct {
	// Source selects the administrator-managed repository connection.
	Source string
	// Path identifies the file within the configured repository.
	Path string
	// Start is the inclusive first original line, or zero for the complete file.
	Start int
	// End is the inclusive final original line, or zero for the complete file.
	End int
	// Notes contains bounded annotations keyed to original source lines.
	Notes []annotation
	// ReferencePosition optionally overrides the configured annotation gutter side.
	ReferencePosition string
	// ReferenceColor optionally overrides the configured annotation color.
	ReferenceColor string
	// HighlightReferences optionally overrides referenced-line highlighting.
	HighlightReferences optionalBool
	// ShowLineNumbers optionally overrides source line-number visibility.
	ShowLineNumbers optionalBool
	// ShowProvider optionally overrides the provider badge visibility.
	ShowProvider optionalBool
	// ShowBranch optionally overrides the revision badge visibility.
	ShowBranch optionalBool
	// Invalid prevents malformed matching invocations from performing capability calls.
	Invalid bool
}

// resourceReader loads one structured setting record owned by this plugin.
type resourceReader func(resource, key string) (sdk.PluginResourceRecord, error)

// settingsReader loads one plugin-owned singleton setting through the generic settings namespace.
type settingsReader func(key string) (sdk.StoredValue, error)

// httpDoer performs one generic host-mediated HTTP request.
type httpDoer func(sdk.HTTPRequest) (sdk.HTTPResponse, error)

// parse accepts quoted macro attributes and marks invalid matching invocations without fetching.
func parse(line string) (options, bool) {
	value := strings.TrimSpace(line)
	body, ok := strings.CutPrefix(value, "{{external-file ")
	if !ok {
		return options{}, false
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok {
		return options{}, false
	}
	invalid := options{Invalid: true}
	if len(body) > 16384 {
		return invalid, true
	}

	result := options{}
	seen := make(map[string]bool)
	for strings.TrimSpace(body) != "" {
		name, argument, remaining, parsed := macroargs.NextQuoted(body)
		if !parsed {
			return invalid, true
		}
		if name != "note" && seen[name] {
			return invalid, true
		}
		seen[name] = true
		body = remaining

		if !applyOption(&result, name, argument) {
			return invalid, true
		}
	}
	if result.Source == "" || result.Path == "" || len(result.Source) > 128 || len(result.Path) > 1024 {
		return invalid, true
	}

	return result, true
}

// applyOption validates and stores one parsed external-file macro attribute.
func applyOption(result *options, name, argument string) bool {
	switch name {
	case "source":
		result.Source = argument
	case "path":
		result.Path = argument
	case "lines":
		start, end, ok := parseLineRange(argument)
		if !ok {
			return false
		}
		result.Start, result.End = start, end
	case "note":
		note, ok := parseAnnotation(argument)
		if !ok || len(result.Notes) >= maxAnnotations {
			return false
		}
		result.Notes = append(result.Notes, note)
	case "reference-position":
		if !validReferencePosition(argument) {
			return false
		}
		result.ReferencePosition = argument
	case "reference-color":
		if !validReferenceColor(argument) {
			return false
		}
		result.ReferenceColor = argument
	case "highlight-references":
		value, ok := parseOptionalBool(argument)
		if !ok {
			return false
		}
		result.HighlightReferences = value
	case "line-numbers":
		value, ok := parseOptionalBool(argument)
		if !ok {
			return false
		}
		result.ShowLineNumbers = value
	case "show-provider":
		value, ok := parseOptionalBool(argument)
		if !ok {
			return false
		}
		result.ShowProvider = value
	case "show-branch":
		value, ok := parseOptionalBool(argument)
		if !ok {
			return false
		}
		result.ShowBranch = value
	default:
		return false
	}

	return true
}

// parseLineRange parses one inclusive one-based line or line range.
func parseLineRange(value string) (int, int, bool) {
	start, finish, rangeFound := strings.Cut(value, "-")
	if !rangeFound {
		finish = start
	}
	first, err := strconv.Atoi(start)
	if err != nil {
		return 0, 0, false
	}
	last, err := strconv.Atoi(finish)
	if err != nil || first < 1 || last < first || last > maxLines {
		return 0, 0, false
	}
	return first, last, true
}

// parseAnnotation parses one line-or-range:description note declaration.
func parseAnnotation(value string) (annotation, bool) {
	selection, description, found := strings.Cut(value, ":")
	if !found || strings.TrimSpace(description) == "" || len(description) > maxAnnotationBytes {
		return annotation{}, false
	}
	start, end, ok := parseLineRange(selection)
	if !ok {
		return annotation{}, false
	}
	return annotation{Start: start, End: end, Text: description}, true
}

// parseOptionalBool parses one explicit boolean macro override.
func parseOptionalBool(value string) (optionalBool, bool) {
	switch value {
	case "true":
		return optionalBool{Set: true, Value: true}, true
	case "false":
		return optionalBool{Set: true, Value: false}, true
	default:
		return optionalBool{}, false
	}
}
