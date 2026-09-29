package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"strings"
	"unicode/utf8"

	"github.com/kumbuka-me/plugins/internal/ascii"
	"github.com/kumbuka-me/plugins/internal/macroargs"

	sdk "github.com/kumbuka-me/sdk"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
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

var namedStatusColors = map[string]string{
	"gray":   "#64748b",
	"blue":   "#2563eb",
	"green":  "#16a34a",
	"yellow": "#ca8a04",
	"orange": "#ea580c",
	"red":    "#dc2626",
	"purple": "#9333ea",
	"teal":   "#0f766e",
	"pink":   "#db2777",
}

var defaultStatusColors = []string{
	"#64748b",
	"#2563eb",
	"#ca8a04",
	"#16a34a",
	"#9333ea",
	"#0f766e",
	"#ea580c",
	"#db2777",
}

// statusOptions contains one parsed status declaration from page Markdown.
type statusOptions struct {
	// ID is the globally stable storage identity for this status instance.
	ID string
	// Set selects a built-in or administrator-managed reusable status set.
	Set string
	// Choices contains page-local choices when the declaration uses options.
	Choices []statusChoice
	// Initial optionally selects the initial value instead of the set's first status.
	Initial string
	// Prefix is optional text shown before the current status value.
	Prefix string
	// Style is either solid or outline.
	Style string
}

// statusChoice is one allowed label and color in a status set.
type statusChoice struct {
	// Label is the value persisted for this choice.
	Label string `json:"label"`
	// Color is a validated six-digit hexadecimal presentation color.
	Color string `json:"color"`
}

// statusSet is one validated reusable or page-local set of status choices.
type statusSet struct {
	// Name is the configured resource key, built-in set name, or inline marker.
	Name string
	// Choices contains ordered allowed values; the first is the default.
	Choices []statusChoice
}

// resourceReader loads one administrator-managed resource record.
type resourceReader func(resource, key string) (sdk.PluginResourceRecord, error)

// storageReader reads one plugin-owned persisted status value.
type storageReader func(key string) (sdk.StoredValue, error)

// transformSource renders status declarations outside fenced and inline code, including visible configuration errors.
func transformSource(source string, readResource resourceReader, readStorage storageReader, localizer sdk.Localizer) string {
	lines := strings.Split(source, "\n")
	sets := make(map[string]statusSet)
	var output strings.Builder
	fence := ""
	count := 0

	for index, line := range lines {
		if index > 0 {
			output.WriteByte('\n')
		}

		if fence != "" {
			output.WriteString(line)
			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}
			continue
		}

		if marker := pluginmarkdown.Fence(line); marker != "" {
			fence = marker
			output.WriteString(line)
			continue
		}

		transformed, used := transformLine(line, maxStatusDeclarations-count, sets, readResource, readStorage, localizer)
		count += used
		output.WriteString(transformed)
	}

	return output.String()
}

// transformLine renders status declarations on one non-fenced Markdown line while preserving inline code spans.
func transformLine(line string, remaining int, sets map[string]statusSet, readResource resourceReader, readStorage storageReader, localizer sdk.Localizer) (string, int) {
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
			relativeEnd := strings.Index(line[index:], "}}")
			if relativeEnd < 0 {
				output.WriteString(statusErrorHTML("unterminated status declaration", localizer))
				used++
				break
			}

			end := index + relativeEnd + 2
			token := line[index:end]
			if len(token) > maxStatusTokenBytes {
				output.WriteString(statusErrorHTML("status declaration is too long", localizer))
				used++
				index = end
				continue
			}

			options, err := parseStatusToken(token)
			if err != nil {
				output.WriteString(statusErrorHTML(err.Error(), localizer))
			} else {
				output.WriteString(renderStatus(options, sets, readResource, readStorage, localizer))
			}
			used++
			index = end
			continue
		}

		output.WriteByte(line[index])
		index++
	}

	return output.String(), used
}

// parsedStatusAttributes contains declaration attributes before page-local choices are expanded.
type parsedStatusAttributes struct {
	// Options contains normalized scalar status options.
	Options statusOptions
	// LocalOptions contains page-local labels when no reusable set is selected.
	LocalOptions string
	// LocalColors contains optional colors parallel to LocalOptions.
	LocalColors string
}

// parseStatusToken parses one complete {{status ...}} declaration and returns a user-facing configuration error.
func parseStatusToken(token string) (statusOptions, error) {
	body, err := statusTokenBody(token)
	if err != nil {
		return statusOptions{}, err
	}
	attributes, err := parseStatusAttributes(body)
	if err != nil {
		return statusOptions{}, err
	}
	if err := validateStatusAttributes(attributes); err != nil {
		return statusOptions{}, err
	}

	if attributes.LocalOptions != "" {
		choices, err := parseInlineChoices(attributes.LocalOptions, attributes.LocalColors)
		if err != nil {
			return statusOptions{}, err
		}
		attributes.Options.Choices = choices
	}
	return attributes.Options, nil
}

// statusTokenBody validates the declaration wrapper and returns its attribute body.
func statusTokenBody(token string) (string, error) {
	value := strings.TrimSpace(token)
	body, ok := strings.CutPrefix(value, "{{status")
	if !ok || len(value) > maxStatusTokenBytes {
		return "", fmt.Errorf("invalid status declaration")
	}
	body, ok = strings.CutSuffix(body, "}}")
	if !ok || body == "" || body[0] != ' ' && body[0] != '\t' {
		return "", fmt.Errorf("status attributes are required")
	}
	return body, nil
}

// parseStatusAttributes consumes unique quoted attributes into their typed destinations.
func parseStatusAttributes(body string) (parsedStatusAttributes, error) {
	result := parsedStatusAttributes{Options: statusOptions{Style: "solid"}}
	seen := make(map[string]bool)
	for strings.TrimSpace(body) != "" {
		name, argument, remaining, parsed := macroargs.NextQuoted(body)
		if !parsed {
			return parsedStatusAttributes{}, fmt.Errorf("invalid status attribute syntax")
		}
		if seen[name] {
			return parsedStatusAttributes{}, fmt.Errorf("duplicate status attribute %q", name)
		}
		seen[name] = true
		body = remaining

		if !applyStatusAttribute(&result, name, argument) {
			return parsedStatusAttributes{}, fmt.Errorf("unknown status attribute %q", name)
		}
	}
	return result, nil
}

// applyStatusAttribute assigns one supported status declaration attribute.
func applyStatusAttribute(result *parsedStatusAttributes, name, argument string) bool {
	switch name {
	case "id":
		result.Options.ID = argument
	case "set":
		result.Options.Set = argument
	case "options":
		result.LocalOptions = argument
	case "colors":
		result.LocalColors = argument
	case "initial":
		result.Options.Initial = argument
	case "prefix":
		result.Options.Prefix = argument
	case "style":
		result.Options.Style = argument
	default:
		return false
	}
	return true
}

// validateStatusAttributes checks scalar bounds and mutually exclusive declaration modes.
func validateStatusAttributes(attributes parsedStatusAttributes) error {
	options := attributes.Options
	switch {
	case !validName(options.ID, maxStatusIDBytes):
		return fmt.Errorf("status id is missing or invalid")
	case options.Set != "" && !validName(options.Set, maxSetNameBytes):
		return fmt.Errorf("status set name is invalid")
	case len(options.Initial) > maxStatusLabelBytes:
		return fmt.Errorf("status initial value is too long")
	case len(options.Prefix) > maxPrefixBytes:
		return fmt.Errorf("status prefix is too long")
	case options.Style != "solid" && options.Style != "outline":
		return fmt.Errorf("status style must be solid or outline")
	case options.Set != "" && attributes.LocalOptions != "":
		return fmt.Errorf("status must use either set or options, not both")
	case options.Set == "" && attributes.LocalOptions == "":
		return fmt.Errorf("status requires options or a reusable set")
	case attributes.LocalColors != "" && attributes.LocalOptions == "":
		return fmt.Errorf("status colors require page-local options")
	default:
		return nil
	}
}

// parseInlineChoices parses page-local labels and optional parallel colors from a status declaration.
func parseInlineChoices(options, colors string) ([]statusChoice, error) {
	labels := splitStatusList(options)
	if len(labels) == 0 || len(labels) > maxStatusOptions {
		return nil, fmt.Errorf("status options must contain between 1 and %d values", maxStatusOptions)
	}

	colorValues := splitStatusList(colors)
	if colors != "" && len(colorValues) != len(labels) {
		return nil, fmt.Errorf("status colors must contain exactly one value per option")
	}

	seen := make(map[string]bool, len(labels))
	choices := make([]statusChoice, 0, len(labels))
	for index, label := range labels {
		if label == "" || len(label) > maxStatusLabelBytes || !utf8.ValidString(label) {
			return nil, fmt.Errorf("status option %d is empty or too long", index+1)
		}
		if seen[label] {
			return nil, fmt.Errorf("status option %q is duplicated", label)
		}
		seen[label] = true

		color := defaultStatusColors[index%len(defaultStatusColors)]
		if colors != "" {
			var ok bool
			color, ok = normalizeStatusColor(colorValues[index])
			if !ok {
				return nil, fmt.Errorf("status color %q is invalid", colorValues[index])
			}
		}
		choices = append(choices, statusChoice{Label: label, Color: color})
	}
	return choices, nil
}

// splitStatusList splits a compact page-local list, preferring semicolons so labels may contain commas.
func splitStatusList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	separator := ","
	if strings.Contains(value, ";") {
		separator = ";"
	}
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		result = append(result, strings.TrimSpace(part))
	}
	return result
}

// validName validates bounded ASCII identifiers used for status IDs and set names.
func validName(value string, maxBytes int) bool {
	if value == "" || len(value) > maxBytes {
		return false
	}
	for index := 0; index < len(value); index++ {
		char := value[index]
		if ascii.IsAlphaNumeric(char) || char == '-' || char == '_' || char == '.' || char == '/' || char == ':' {
			continue
		}
		return false
	}
	return true
}

// resolveStatusSet returns page-local choices directly or loads one reusable status set.
func resolveStatusSet(options statusOptions, sets map[string]statusSet, read resourceReader) (statusSet, error) {
	if len(options.Choices) > 0 {
		return statusSet{Name: "inline", Choices: options.Choices}, nil
	}
	if set, ok := sets[options.Set]; ok {
		return set, nil
	}
	set, err := loadStatusSet(options.Set, read)
	if err != nil {
		return statusSet{}, err
	}
	sets[options.Set] = set
	return set, nil
}

// loadStatusSet resolves an administrator-managed set first and then falls back to built-in sets.
func loadStatusSet(name string, read resourceReader) (statusSet, error) {
	if read != nil {
		record, err := read("sets", name)
		if err == nil {
			set, parseErr := statusSetFromRecord(name, record)
			if parseErr != nil {
				return statusSet{}, fmt.Errorf("invalid status set %q: %w", name, parseErr)
			}
			return set, nil
		}
	}
	if set, ok := builtinStatusSet(name); ok {
		return set, nil
	}
	return statusSet{}, fmt.Errorf("unknown status set %q", name)
}

// statusSetFromRecord parses and validates one administrator-managed structured status set.
func statusSetFromRecord(name string, record sdk.PluginResourceRecord) (statusSet, error) {
	if record.Key != "" && record.Key != name {
		return statusSet{}, fmt.Errorf("record key does not match the requested set")
	}

	source := strings.TrimSpace(record.Values["statuses"])
	if source == "" {
		return statusSet{}, fmt.Errorf("no statuses are configured")
	}
	if strings.HasPrefix(source, "[") {
		var choices []statusChoice
		if err := json.Unmarshal([]byte(source), &choices); err != nil {
			return statusSet{}, fmt.Errorf("structured statuses are invalid")
		}
		return validateStatusChoices(name, choices)
	}

	return parseLegacyStatusSet(name, source)
}

// parseLegacyStatusSet accepts pre-structured Label|color records so existing sets remain readable after upgrade.
func parseLegacyStatusSet(name, source string) (statusSet, error) {
	choices := make([]statusChoice, 0)
	for row, rawLine := range strings.Split(source, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		label, colorName, found := strings.Cut(line, "|")
		if !found {
			return statusSet{}, fmt.Errorf("row %d must contain Label|color", row+1)
		}
		color, ok := normalizeStatusColor(colorName)
		if !ok {
			return statusSet{}, fmt.Errorf("row %d uses invalid color %q", row+1, strings.TrimSpace(colorName))
		}
		choices = append(choices, statusChoice{Label: strings.TrimSpace(label), Color: color})
	}
	return validateStatusChoices(name, choices)
}

// validateStatusChoices validates labels, colors, duplicates, and option count for one status set.
func validateStatusChoices(name string, choices []statusChoice) (statusSet, error) {
	if len(choices) == 0 || len(choices) > maxStatusOptions {
		return statusSet{}, fmt.Errorf("must contain between 1 and %d statuses", maxStatusOptions)
	}

	seen := make(map[string]bool, len(choices))
	result := statusSet{Name: name, Choices: make([]statusChoice, 0, len(choices))}
	for index, choice := range choices {
		choice.Label = strings.TrimSpace(choice.Label)
		if choice.Label == "" || len(choice.Label) > maxStatusLabelBytes || !utf8.ValidString(choice.Label) {
			return statusSet{}, fmt.Errorf("row %d has an empty or too-long label", index+1)
		}
		if seen[choice.Label] {
			return statusSet{}, fmt.Errorf("row %d duplicates status %q", index+1, choice.Label)
		}
		seen[choice.Label] = true

		color, ok := normalizeStatusColor(choice.Color)
		if !ok {
			return statusSet{}, fmt.Errorf("row %d uses invalid color %q", index+1, choice.Color)
		}
		choice.Color = color
		result.Choices = append(result.Choices, choice)
	}
	return result, nil
}

// builtinStatusSet returns one first-party reusable status set that works without administration setup.
func builtinStatusSet(name string) (statusSet, bool) {
	switch name {
	case "workflow":
		return statusSet{Name: name, Choices: []statusChoice{
			{Label: "To do", Color: namedStatusColors["gray"]},
			{Label: "In progress", Color: namedStatusColors["blue"]},
			{Label: "Blocked", Color: namedStatusColors["red"]},
			{Label: "Done", Color: namedStatusColors["green"]},
		}}, true
	case "approval":
		return statusSet{Name: name, Choices: []statusChoice{
			{Label: "Draft", Color: namedStatusColors["gray"]},
			{Label: "In review", Color: namedStatusColors["yellow"]},
			{Label: "Approved", Color: namedStatusColors["green"]},
			{Label: "Rejected", Color: namedStatusColors["red"]},
		}}, true
	default:
		return statusSet{}, false
	}
}

// normalizeStatusColor converts supported named colors or six-digit hexadecimal colors to canonical hexadecimal form.
func normalizeStatusColor(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	if color, ok := namedStatusColors[value]; ok {
		return color, true
	}
	if len(value) != 7 || value[0] != '#' {
		return "", false
	}
	for index := 1; index < len(value); index++ {
		if !ascii.IsLowerHex(value[index]) {
			return "", false
		}
	}
	return value, true
}

// selectedChoice resolves persisted state, the declaration initial value, and finally the set default.
func selectedChoice(options statusOptions, set statusSet, read storageReader) (statusChoice, error) {
	selected := set.Choices[0].Label
	if options.Initial != "" {
		if _, ok := findChoice(set, options.Initial); !ok {
			return statusChoice{}, fmt.Errorf("initial status %q is not one of the configured options", options.Initial)
		}
		selected = options.Initial
	}
	if read != nil {
		stored, err := read(storageKey(options.ID))
		if err == nil && stored.Found && len(stored.Value) <= maxStatusLabelBytes {
			if _, ok := findChoice(set, string(stored.Value)); ok {
				selected = string(stored.Value)
			}
		}
	}
	choice, _ := findChoice(set, selected)
	return choice, nil
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

// renderStatus resolves one status declaration into safe inline HTML or an explicit inline error.
func renderStatus(options statusOptions, sets map[string]statusSet, readResource resourceReader, readStorage storageReader, localizer sdk.Localizer) string {
	set, err := resolveStatusSet(options, sets, readResource)
	if err != nil {
		return statusErrorHTML(err.Error(), localizer)
	}
	choice, err := selectedChoice(options, set, readStorage)
	if err != nil {
		return statusErrorHTML(err.Error(), localizer)
	}
	return statusHTML(options, set, choice, localizer)
}

// statusErrorHTML renders one escaped inline configuration error instead of silently hiding a broken status.
func statusErrorHTML(message string, localizer sdk.Localizer) string {
	escaped := html.EscapeString(message)
	return `<span class="kumbuka-status kumbuka-status-error" title="` + escaped + `">` + html.EscapeString(localizer.Textf("status.error", message)) + `</span>`
}

// statusHTML renders one escaped fallback badge plus bounded browser-module choice metadata.
func statusHTML(options statusOptions, set statusSet, choice statusChoice, localizer sdk.Localizer) string {
	var output strings.Builder
	output.WriteString(`<span class="kumbuka-status-browser" data-kumbuka-plugin="me.kumbuka.status-dropdowns" data-kumbuka-module="status-ui" data-kumbuka-input="html">`)
	output.WriteString(`<span class="kumbuka-status-fallback" data-kumbuka-fallback>`)
	output.WriteString(`<span class="kumbuka-status-options`)
	for index, candidate := range set.Choices {
		output.WriteString(` kumbuka-status-choice__`)
		output.WriteString(hex.EncodeToString([]byte(candidate.Color)))
		output.WriteString(`__`)
		output.WriteString(actionID(options.ID, index))
		output.WriteString(`__`)
		output.WriteString(hex.EncodeToString([]byte(statusChoiceLabel(set, candidate, localizer))))
	}
	output.WriteString(`"></span>`)
	if options.Prefix != "" {
		output.WriteString(`<span class="kumbuka-status-prefix">`)
		output.WriteString(html.EscapeString(options.Prefix))
		output.WriteString(`</span>`)
	}
	writeStatusBadge(&output, options, set, choice, localizer)
	output.WriteString(`</span></span>`)
	return output.String()
}

// writeStatusBadge renders the passive fallback shown when browser modules are unavailable.
func writeStatusBadge(output *strings.Builder, options statusOptions, set statusSet, choice statusChoice, localizer sdk.Localizer) {
	output.WriteString(`<span class="kumbuka-status kumbuka-status-`)
	output.WriteString(statusToneForColor(choice.Color))
	output.WriteString(` kumbuka-status-`)
	output.WriteString(options.Style)
	output.WriteString(`" title="`)
	displayLabel := statusChoiceLabel(set, choice, localizer)
	output.WriteString(html.EscapeString(localizer.Textf("status.title", displayLabel)))
	output.WriteString(`">`)
	output.WriteString(`<span class="kumbuka-status-value">`)
	output.WriteString(html.EscapeString(displayLabel))
	output.WriteString(`</span></span>`)
}

// statusChoiceLabel returns a localized display label for built-in status sets while preserving configured labels.
func statusChoiceLabel(set statusSet, choice statusChoice, localizer sdk.Localizer) string {
	keys := map[string]map[string]string{
		"workflow": {
			"To do":       "status.workflow.todo",
			"In progress": "status.workflow.in_progress",
			"Blocked":     "status.workflow.blocked",
			"Done":        "status.workflow.done",
		},
		"approval": {
			"Draft":     "status.approval.draft",
			"In review": "status.approval.in_review",
			"Approved":  "status.approval.approved",
			"Rejected":  "status.approval.rejected",
		},
	}
	if choices := keys[set.Name]; choices != nil {
		if key := choices[choice.Label]; key != "" {
			return localizer.Text(key)
		}
	}
	return choice.Label
}

// statusToneForColor returns a bounded fallback class for known colors and gray for arbitrary custom colors.
func statusToneForColor(color string) string {
	for name, candidate := range namedStatusColors {
		if candidate == color {
			return name
		}
	}
	return "gray"
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
