package main

import (
	"crypto/sha256"
	"encoding/hex"
	"html"
	"strconv"
	"strings"
	"unicode/utf8"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	maxStatusDeclarations = 64
	maxStatusOptions      = 16
	maxStatusTokenBytes   = 2048
	maxStatusIDBytes      = 128
	maxSetNameBytes       = 128
	maxStatusLabelBytes   = 64
	maxPrefixBytes        = 64
)

// statusOptions contains one parsed status declaration from page Markdown.
type statusOptions struct {
	// ID is the globally stable storage identity for this status instance.
	ID string
	// Set selects a built-in or administrator-managed status set.
	Set string
	// Initial optionally selects the initial value instead of the set's first status.
	Initial string
	// Prefix is optional text shown before the current status value.
	Prefix string
	// Style is either solid or outline.
	Style string
}

// statusChoice is one allowed label and visual tone in a status set.
type statusChoice struct {
	// Label is the value persisted for this choice.
	Label string
	// Tone selects one bounded presentation class.
	Tone string
}

// statusSet is one validated reusable set of status choices.
type statusSet struct {
	// Name is the configured resource key or built-in set name.
	Name string
	// Choices contains ordered allowed values; the first is the default.
	Choices []statusChoice
}

// resourceReader loads one administrator-managed resource record.
type resourceReader func(resource, key string) (sdk.PluginResourceRecord, error)

// storageReader reads one plugin-owned persisted status value.
type storageReader func(key string) (sdk.StoredValue, error)

// transformSource renders all valid status declarations outside fenced and inline code.
func transformSource(source string, readResource resourceReader, readStorage storageReader) string {
	lines := strings.Split(source, "\n")
	sets := make(map[string]statusSet)
	var output strings.Builder
	var fence byte
	var fenceLength int
	count := 0

	for index, line := range lines {
		if index > 0 {
			output.WriteByte('\n')
		}

		if fence != 0 {
			output.WriteString(line)
			if closesFence(line, fence, fenceLength) {
				fence = 0
				fenceLength = 0
			}
			continue
		}

		if marker, length := openingFence(line); marker != 0 {
			fence = marker
			fenceLength = length
			output.WriteString(line)
			continue
		}

		transformed, used := transformLine(line, maxStatusDeclarations-count, sets, readResource, readStorage)
		count += used
		output.WriteString(transformed)
	}

	return output.String()
}

// transformLine renders status declarations on one non-fenced Markdown line while preserving inline code spans.
func transformLine(line string, remaining int, sets map[string]statusSet, readResource resourceReader, readStorage storageReader) (string, int) {
	if remaining <= 0 || !strings.Contains(line, "{{status") {
		return line, 0
	}

	var output strings.Builder
	used := 0
	codeTicks := 0
	for index := 0; index < len(line); {
		if line[index] == '`' {
			run := repeatedByte(line[index:], '`')
			output.WriteString(line[index : index+run])
			if codeTicks == 0 {
				codeTicks = run
			} else if run == codeTicks {
				codeTicks = 0
			}
			index += run
			continue
		}

		if codeTicks == 0 && used < remaining && strings.HasPrefix(line[index:], "{{status") {
			end := strings.Index(line[index:], "}}")
			if end >= 0 {
				end += index + 2
				token := line[index:end]
				if len(token) <= maxStatusTokenBytes {
					if options, ok := parseStatusToken(token); ok {
						output.WriteString(renderStatus(options, sets, readResource, readStorage))
						used++
						index = end
						continue
					}
				}
			}
		}

		output.WriteByte(line[index])
		index++
	}

	return output.String(), used
}

// parseStatusToken parses one complete {{status ...}} declaration.
func parseStatusToken(token string) (statusOptions, bool) {
	value := strings.TrimSpace(token)
	body, ok := strings.CutPrefix(value, "{{status")
	if !ok || len(value) > maxStatusTokenBytes {
		return statusOptions{}, false
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok || body == "" || body[0] != ' ' && body[0] != '\t' {
		return statusOptions{}, false
	}

	result := statusOptions{Style: "solid"}
	seen := make(map[string]bool)
	for strings.TrimSpace(body) != "" {
		name, argument, remaining, parsed := parseAttribute(body)
		if !parsed || seen[name] {
			return statusOptions{}, false
		}
		seen[name] = true
		body = remaining

		switch name {
		case "id":
			result.ID = argument
		case "set":
			result.Set = argument
		case "initial":
			result.Initial = argument
		case "prefix":
			result.Prefix = argument
		case "style":
			result.Style = argument
		default:
			return statusOptions{}, false
		}
	}

	if !validName(result.ID, maxStatusIDBytes) || !validName(result.Set, maxSetNameBytes) ||
		len(result.Initial) > maxStatusLabelBytes || len(result.Prefix) > maxPrefixBytes ||
		result.Style != "solid" && result.Style != "outline" {
		return statusOptions{}, false
	}
	return result, true
}

// parseAttribute consumes one quoted name="value" attribute from a declaration body.
func parseAttribute(body string) (name, value, remaining string, ok bool) {
	body = strings.TrimSpace(body)
	equals := strings.IndexByte(body, '=')
	if equals <= 0 {
		return "", "", "", false
	}

	name = strings.TrimSpace(body[:equals])
	rest := strings.TrimSpace(body[equals+1:])
	if name == "" || len(rest) < 2 || rest[0] != '"' {
		return "", "", "", false
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
		return "", "", "", false
	}

	argument, err := strconv.Unquote(rest[:end+1])
	if err != nil || !utf8.ValidString(argument) {
		return "", "", "", false
	}
	remaining = rest[end+1:]
	if remaining != "" && remaining[0] != ' ' && remaining[0] != '\t' {
		return "", "", "", false
	}
	return name, argument, remaining, true
}

// validName validates bounded ASCII identifiers used for status IDs and set names.
func validName(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' ||
			char == '-' || char == '_' || char == '.' || char == '/' || char == ':' {
			continue
		}
		return false
	}
	return true
}

// loadStatusSet resolves a custom status set first and then falls back to built-in sets.
func loadStatusSet(name string, read resourceReader) (statusSet, bool) {
	if read != nil {
		record, err := read("sets", name)
		if err == nil {
			if set, ok := statusSetFromRecord(name, record); ok {
				return set, true
			}
		}
	}
	return builtinStatusSet(name)
}

// statusSetFromRecord parses and validates one administrator-managed status set.
func statusSetFromRecord(name string, record sdk.PluginResourceRecord) (statusSet, bool) {
	if record.Key != "" && record.Key != name {
		return statusSet{}, false
	}
	return parseStatusSet(name, record.Values["statuses"])
}

// parseStatusSet parses newline-separated Label|tone status choices.
func parseStatusSet(name, source string) (statusSet, bool) {
	result := statusSet{Name: name}
	seen := make(map[string]bool)
	for _, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		label, tone, found := strings.Cut(line, "|")
		label = strings.TrimSpace(label)
		tone = strings.ToLower(strings.TrimSpace(tone))
		if !found || label == "" || len(label) > maxStatusLabelBytes || seen[label] || !validTone(tone) || len(result.Choices) >= maxStatusOptions {
			return statusSet{}, false
		}
		seen[label] = true
		result.Choices = append(result.Choices, statusChoice{Label: label, Tone: tone})
	}
	return result, len(result.Choices) > 0
}

// builtinStatusSet returns one first-party status set that works without administration setup.
func builtinStatusSet(name string) (statusSet, bool) {
	switch name {
	case "workflow":
		return statusSet{Name: name, Choices: []statusChoice{
			{Label: "To do", Tone: "gray"},
			{Label: "In progress", Tone: "blue"},
			{Label: "Blocked", Tone: "red"},
			{Label: "Done", Tone: "green"},
		}}, true
	case "approval":
		return statusSet{Name: name, Choices: []statusChoice{
			{Label: "Draft", Tone: "gray"},
			{Label: "In review", Tone: "yellow"},
			{Label: "Approved", Tone: "green"},
			{Label: "Rejected", Tone: "red"},
		}}, true
	default:
		return statusSet{}, false
	}
}

// validTone reports whether a status tone has a corresponding bounded CSS class.
func validTone(tone string) bool {
	switch tone {
	case "gray", "blue", "green", "yellow", "orange", "red", "purple", "teal":
		return true
	default:
		return false
	}
}

// selectedChoice resolves persisted state, the declaration initial value, and finally the set default.
func selectedChoice(options statusOptions, set statusSet, read storageReader) statusChoice {
	selected := options.Initial
	if selected == "" && len(set.Choices) > 0 {
		selected = set.Choices[0].Label
	}
	if read != nil {
		stored, err := read(storageKey(options.ID))
		if err == nil && stored.Found && len(stored.Value) <= maxStatusLabelBytes {
			selected = string(stored.Value)
		}
	}
	if choice, ok := findChoice(set, selected); ok {
		return choice
	}
	return set.Choices[0]
}

// findChoice returns the configured choice with the requested exact label.
func findChoice(set statusSet, label string) (statusChoice, bool) {
	for _, choice := range set.Choices {
		if choice.Label == label {
			return choice, true
		}
	}
	return statusChoice{}, false
}

// renderStatus resolves one status declaration into safe inline HTML.
func renderStatus(options statusOptions, sets map[string]statusSet, readResource resourceReader, readStorage storageReader) string {
	set, ok := sets[options.Set]
	if !ok {
		set, ok = loadStatusSet(options.Set, readResource)
		if ok {
			sets[options.Set] = set
		}
	}
	if !ok || len(set.Choices) == 0 {
		return `<span class="kumbuka-status kumbuka-status-error" title="Unknown or invalid status set">Status unavailable</span>`
	}

	choice := selectedChoice(options, set, readStorage)
	return statusHTML(options, choice)
}

// statusHTML renders one escaped status badge with bounded class names.
func statusHTML(options statusOptions, choice statusChoice) string {
	var output strings.Builder
	output.WriteString(`<span class="kumbuka-status kumbuka-status-`)
	output.WriteString(choice.Tone)
	output.WriteString(` kumbuka-status-`)
	output.WriteString(options.Style)
	output.WriteString(`" title="Status: `)
	output.WriteString(html.EscapeString(choice.Label))
	output.WriteString(`">`)
	if options.Prefix != "" {
		output.WriteString(`<span class="kumbuka-status-prefix">`)
		output.WriteString(html.EscapeString(options.Prefix))
		output.WriteString(`</span>`)
	}
	output.WriteString(`<span class="kumbuka-status-value">`)
	output.WriteString(html.EscapeString(choice.Label))
	output.WriteString(`</span></span>`)
	return output.String()
}

// storageKey derives a bounded opaque storage key from a public status identifier.
func storageKey(id string) string {
	sum := sha256.Sum256([]byte(id))
	return "status." + hex.EncodeToString(sum[:16])
}

// repeatedByte returns the length of the leading run of the requested byte.
func repeatedByte(value string, target byte) int {
	count := 0
	for count < len(value) && value[count] == target {
		count++
	}
	return count
}

// openingFence recognizes a top-level CommonMark-style backtick or tilde fence.
func openingFence(line string) (byte, int) {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" || trimmed[0] != '`' && trimmed[0] != '~' {
		return 0, 0
	}
	run := repeatedByte(trimmed, trimmed[0])
	if run < 3 {
		return 0, 0
	}
	return trimmed[0], run
}

// closesFence reports whether a line closes the active top-level fence.
func closesFence(line string, marker byte, minimum int) bool {
	trimmed := strings.TrimLeft(line, " ")
	if len(line)-len(trimmed) > 3 || trimmed == "" || trimmed[0] != marker {
		return false
	}
	run := repeatedByte(trimmed, marker)
	return run >= minimum && strings.TrimSpace(trimmed[run:]) == ""
}
