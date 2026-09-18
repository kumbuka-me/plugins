// Package main implements external repository file embeds using generic Kumbuka plugin capabilities.
package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

// main is the WASI package entry point.
func main() {}

// init registers the external-file macro with the Kumbuka SDK.
func init() {
	sdk.RegisterMacro("external-files", parse, func(value options) (sdk.Result, error) {
		return render(value, sdk.Resources().Get, sdk.HTTP().Do), nil
	})
}

// annotation describes one numbered note attached to an original source line.
type annotation struct {
	Line int
	Text string
}

// options contains one parsed external-file macro invocation.
type options struct {
	Source, Path string
	Start, End   int
	Notes        []annotation
	Invalid      bool
}

// resourceReader loads one structured setting record owned by this plugin.
type resourceReader func(resource, key string) (sdk.PluginResourceRecord, error)

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

	result := options{}
	invalid := func() (options, bool) { return options{Invalid: true}, true }
	if len(body) > 16384 {
		return invalid()
	}

	seen := map[string]bool{}
	for strings.TrimSpace(body) != "" {
		body = strings.TrimSpace(body)
		name, rest, found := strings.Cut(body, "=")
		if !found {
			return invalid()
		}
		name = strings.TrimSpace(name)
		rest = strings.TrimSpace(rest)
		if len(rest) < 2 || rest[0] != '"' {
			return invalid()
		}

		end := 1
		for end < len(rest) {
			if rest[end] == '\\' {
				end += 2
				continue
			}
			if rest[end] == '"' {
				break
			}
			end++
		}
		if end >= len(rest) {
			return invalid()
		}

		argument, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			return invalid()
		}
		body = rest[end+1:]
		if body != "" && body[0] != ' ' && body[0] != '\t' {
			return invalid()
		}
		if seen[name] && name != "note" {
			return invalid()
		}
		seen[name] = true

		switch name {
		case "source":
			result.Source = argument
		case "path":
			result.Path = argument
		case "lines":
			start, finish, rangeFound := strings.Cut(argument, "-")
			if !rangeFound {
				finish = start
			}
			result.Start, err = strconv.Atoi(start)
			if err != nil {
				return invalid()
			}
			result.End, err = strconv.Atoi(finish)
			if err != nil || result.Start < 1 || result.End < result.Start || result.End > maxLines {
				return invalid()
			}
		case "note":
			lineNumber, description, noteFound := strings.Cut(argument, ":")
			if !noteFound {
				return invalid()
			}
			number, numberErr := strconv.Atoi(lineNumber)
			if numberErr != nil || number < 1 || number > maxLines || strings.TrimSpace(description) == "" || len(description) > maxAnnotationBytes || len(result.Notes) >= maxAnnotations {
				return invalid()
			}
			result.Notes = append(result.Notes, annotation{Line: number, Text: description})
		default:
			return invalid()
		}
	}
	if result.Source == "" || result.Path == "" || len(result.Source) > 128 || len(result.Path) > 1024 {
		return invalid()
	}
	return result, true
}

// message returns one escaped user-facing external-file error box.
func message(text string) sdk.Result {
	return sdk.Text(`<div class="external-file-error">` + html.EscapeString(text) + `</div>`)
}

// render loads the configured source, fetches the file, selects lines, and renders escaped text.
func render(value options, resources resourceReader, httpDo httpDoer) sdk.Result {
	if value.Invalid || value.Source == "" || value.Path == "" || len(value.Notes) > maxAnnotations {
		return message("Invalid external file. Use source and path, optional lines, and note=\"line:description\".")
	}

	source, err := loadSource(value.Source, resources)
	if err != nil || !allowSourceFetch(value.Source) {
		return message("External file unavailable. Ask an administrator to check the configured source and provider access.")
	}
	content, err := fetchFile(source, value.Path, httpDo)
	if err != nil {
		return message("External file unavailable. Ask an administrator to check the configured source and provider access.")
	}
	file, err := selectLines(content, value.Start, value.End)
	if err != nil {
		return message("External file unavailable. Check the requested line range.")
	}
	for _, note := range value.Notes {
		if note.Line < file.Start || note.Line >= file.Start+len(strings.Split(file.Content, "\n")) || len(note.Text) > maxAnnotationBytes {
			return message("An annotation refers to a line outside the displayed file range.")
		}
	}

	var output strings.Builder
	output.WriteString(`<div class="external-file"><div class="external-file-title">` + html.EscapeString(value.Source+": "+value.Path) + `</div><pre class="external-file-code"><code>`)
	markers := make(map[int][]int)
	for index, note := range value.Notes {
		markers[note.Line] = append(markers[note.Line], index+1)
	}
	lines := strings.Split(file.Content, "\n")
	for index, line := range lines {
		number := file.Start + index
		output.WriteString(`<span class="external-file-line"><span class="external-file-number">` + strconv.Itoa(number) + `</span>`)
		for _, marker := range markers[number] {
			fmt.Fprintf(&output, `<span class="external-file-marker" title="Annotation %d">[%d]</span>`, marker, marker)
		}
		output.WriteString(html.EscapeString(line))
		output.WriteString("</span>")
		if index+1 < len(lines) {
			output.WriteByte('\n')
		}
	}
	output.WriteString(`</code></pre>`)
	if len(value.Notes) > 0 {
		output.WriteString(`<div class="external-file-notes"><ol>`)
		for index, note := range value.Notes {
			fmt.Fprintf(&output, `<li><strong>[%d] Line %d:</strong> %s</li>`, index+1, note.Line, html.EscapeString(note.Text))
		}
		output.WriteString(`</ol></div>`)
	}
	output.WriteString(`</div>`)
	return sdk.Text(output.String())
}
