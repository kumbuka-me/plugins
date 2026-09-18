// Package main implements approved external repository file embeds.
package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

func main() {}
func init() {
	sdk.RegisterMacro("external-files", parse, func(o options) (sdk.Result, error) { return render(o, sdk.ExternalFiles().Read), nil })
}

type annotation struct {
	Line int
	Text string
}
type options struct {
	Source, Path string
	Start, End   int
	Notes        []annotation
	Invalid      bool
}
type reader func(sdk.ExternalFileRequest) (sdk.ExternalFile, error)

// parse accepts quoted attributes; note may repeat. Invalid matching macros
// become a helpful error box and never invoke the host capability.
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
	o := options{}
	bad := func() (options, bool) { return options{Invalid: true}, true }
	if len(body) > 16384 {
		return bad()
	}
	seen := map[string]bool{}
	for strings.TrimSpace(body) != "" {
		body = strings.TrimSpace(body)
		name, rest, found := strings.Cut(body, "=")
		if !found {
			return bad()
		}
		name = strings.TrimSpace(name)
		rest = strings.TrimSpace(rest)
		if len(rest) < 2 || rest[0] != '"' {
			return bad()
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
			return bad()
		}
		v, err := strconv.Unquote(rest[:end+1])
		if err != nil {
			return bad()
		}
		body = rest[end+1:]
		if body != "" && body[0] != ' ' && body[0] != '\t' {
			return bad()
		}
		if seen[name] && name != "note" {
			return bad()
		}
		seen[name] = true
		switch name {
		case "source":
			o.Source = v
		case "path":
			o.Path = v
		case "lines":
			a, b, rangeFound := strings.Cut(v, "-")
			if !rangeFound {
				b = a
			}
			o.Start, err = strconv.Atoi(a)
			if err != nil {
				return bad()
			}
			o.End, err = strconv.Atoi(b)
			if err != nil || o.Start < 1 || o.End < o.Start || o.End > 10000 {
				return bad()
			}
		case "note":
			n, text, found := strings.Cut(v, ":")
			if !found {
				return bad()
			}
			number, err := strconv.Atoi(n)
			if err != nil || number < 1 || number > 10000 || strings.TrimSpace(text) == "" || len(text) > 2048 || len(o.Notes) >= 32 {
				return bad()
			}
			o.Notes = append(o.Notes, annotation{number, text})
		default:
			return bad()
		}
	}
	if o.Source == "" || o.Path == "" || len(o.Source) > 64 || len(o.Path) > 1024 {
		return bad()
	}
	return o, true
}

func message(text string) sdk.Result {
	return sdk.Text(`<div class="external-file-error">` + html.EscapeString(text) + `</div>`)
}
func render(o options, read reader) sdk.Result {
	if o.Invalid || o.Source == "" || o.Path == "" || len(o.Notes) > 32 {
		return message("Invalid external file. Use source and path, optional lines, and note=\"line:description\".")
	}
	file, err := read(sdk.ExternalFileRequest{Source: o.Source, Path: o.Path, Start: o.Start, End: o.End})
	if err != nil {
		return message("External file unavailable. Ask an administrator to check source approval, provider access, or the requested lines.")
	}
	if len(file.Content) > 128<<10 || file.Start < 1 {
		return message("External file exceeds display limits.")
	}
	lines := strings.Split(file.Content, "\n")
	if len(lines) > 10000 {
		return message("External file exceeds display limits.")
	}
	for _, n := range o.Notes {
		if n.Line < file.Start || n.Line >= file.Start+len(lines) || len(n.Text) > 2048 {
			return message("An annotation refers to a line outside the displayed file range.")
		}
	}
	var b strings.Builder
	b.WriteString(`<div class="external-file"><div class="external-file-title">` + html.EscapeString(o.Source+": "+o.Path) + `</div><pre class="external-file-code"><code>`)
	markers := map[int][]int{}
	for i, n := range o.Notes {
		markers[n.Line] = append(markers[n.Line], i+1)
	}
	for i, line := range lines {
		number := file.Start + i
		b.WriteString(`<span class="external-file-line"><span class="external-file-number">` + strconv.Itoa(number) + `</span>`)
		for _, marker := range markers[number] {
			fmt.Fprintf(&b, `<span class="external-file-marker" title="Annotation %d">[%d]</span>`, marker, marker)
		}
		b.WriteString(html.EscapeString(line))
		b.WriteString("</span>")
		if i+1 < len(lines) {
			b.WriteByte('\n')
		}
	}
	b.WriteString(`</code></pre>`)
	if len(o.Notes) > 0 {
		b.WriteString(`<div class="external-file-notes"><ol>`)
		for i, n := range o.Notes {
			fmt.Fprintf(&b, `<li><strong>[%d] Line %d:</strong> %s</li>`, i+1, n.Line, html.EscapeString(n.Text))
		}
		b.WriteString(`</ol></div>`)
	}
	b.WriteString(`</div>`)
	return sdk.Text(b.String())
}
