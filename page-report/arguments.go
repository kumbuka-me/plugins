package main

import (
	"cmp"
	"strconv"
	"strings"
)

// parse recognizes one standalone {{pages ...}} invocation.
func parse(line string) (macroOptions, bool) {
	body, ok := pageReportBody(line)
	if !ok {
		return macroOptions{}, false
	}

	arguments, ok := parseArguments(body)
	if !ok {
		return macroOptions{}, false
	}
	return optionsFromArguments(arguments)
}

// pageReportBody extracts the option text from a standalone {{pages ...}} invocation.
func pageReportBody(line string) (string, bool) {
	value := strings.TrimSpace(line)
	body, ok := strings.CutPrefix(value, "{{pages")
	if !ok {
		return "", false
	}

	body, ok = strings.CutSuffix(body, "}}")
	if !ok || (body != "" && body[0] != ' ' && body[0] != '\t') {
		return "", false
	}
	return strings.TrimSpace(body), true
}

// optionsFromArguments normalizes parsed arguments and validates report options.
func optionsFromArguments(arguments map[string]string) (macroOptions, bool) {
	limit, ok := reportLimit(arguments["limit"])
	if !ok {
		return macroOptions{}, false
	}

	options := macroOptions{
		Query:   strings.TrimSpace(arguments["query"]),
		Columns: []string{"title", "status", "owner", "updated"},
		View:    cmp.Or(strings.TrimSpace(arguments["view"]), "table"),
		Sort:    cmp.Or(strings.TrimSpace(arguments["sort"]), "relevance"),
		Limit:   limit,
	}
	if value := strings.TrimSpace(arguments["columns"]); value != "" {
		options.Columns = splitColumns(value)
	}
	if !validOptions(options) {
		return macroOptions{}, false
	}
	return options, true
}

// reportLimit parses an optional bounded report limit.
func reportLimit(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return defaultLimit, true
	}

	limit, err := strconv.Atoi(value)
	return limit, err == nil && limit >= 1 && limit <= maxLimit
}

// validOptions reports whether all required report options and columns are supported.
func validOptions(options macroOptions) bool {
	if options.Query == "" || len(options.Columns) == 0 || !validView(options.View) || !validSort(options.Sort) {
		return false
	}
	for _, column := range options.Columns {
		if !validColumn(column) {
			return false
		}
	}
	return true
}

// validView reports whether view selects a supported page-report presentation.
func validView(view string) bool {
	switch view {
	case "table", "list", "cards":
		return true
	default:
		return false
	}
}

// validSort reports whether sort selects a supported page-report ordering.
func validSort(sort string) bool {
	switch sort {
	case "relevance", "updated", "title", "path":
		return true
	default:
		return false
	}
}

// argumentParser incrementally parses a page-report macro argument list.
type argumentParser struct {
	// value is the complete argument string being parsed.
	value string
	// index is the next unread byte in value.
	index int
}

// parseArguments parses key=value options without using regular expressions.
func parseArguments(value string) (map[string]string, bool) {
	parser := argumentParser{value: value}
	result := map[string]string{}
	for {
		name, value, done, ok := parser.next()
		if !ok {
			return nil, false
		}
		if done {
			return result, true
		}
		if _, exists := result[name]; exists {
			return nil, false
		}
		result[name] = value
	}
}

// next parses one name=value argument or reports the end of the input.
func (p *argumentParser) next() (name, value string, done, ok bool) {
	p.skipSpace()
	if p.index == len(p.value) {
		return "", "", true, true
	}

	name = p.readName()
	if name == "" {
		return "", "", false, false
	}
	p.skipSpace()
	if p.index >= len(p.value) || p.value[p.index] != '=' {
		return "", "", false, false
	}
	p.index++
	p.skipSpace()
	value, ok = p.readValue()
	return name, value, false, ok
}

// skipSpace advances past spaces and horizontal tabs.
func (p *argumentParser) skipSpace() {
	for p.index < len(p.value) && (p.value[p.index] == ' ' || p.value[p.index] == '\t') {
		p.index++
	}
}

// readName consumes one lowercase option name.
func (p *argumentParser) readName() string {
	start := p.index
	for p.index < len(p.value) {
		character := p.value[p.index]
		if (character >= 'a' && character <= 'z') || character == '_' {
			p.index++
			continue
		}
		break
	}
	return p.value[start:p.index]
}

// readValue consumes one quoted or unquoted option value.
func (p *argumentParser) readValue() (string, bool) {
	if p.index >= len(p.value) {
		return "", false
	}
	if p.value[p.index] == '"' {
		return p.readQuotedValue()
	}
	return p.readBareValue()
}

// readBareValue consumes a value up to the next horizontal whitespace.
func (p *argumentParser) readBareValue() (string, bool) {
	start := p.index
	for p.index < len(p.value) && p.value[p.index] != ' ' && p.value[p.index] != '\t' {
		p.index++
	}
	return p.value[start:p.index], p.index > start
}

// readQuotedValue consumes and unquotes a double-quoted option value.
func (p *argumentParser) readQuotedValue() (string, bool) {
	start := p.index
	p.index++
	escaped := false
	for p.index < len(p.value) {
		character := p.value[p.index]
		p.index++
		if escaped {
			escaped = false
			continue
		}
		if character == '\\' {
			escaped = true
			continue
		}
		if character == '"' {
			decoded, err := strconv.Unquote(p.value[start:p.index])
			return decoded, err == nil
		}
	}
	return "", false
}

// splitColumns normalizes a comma-separated report column list.
func splitColumns(value string) []string {
	columns := make([]string, 0)
	for column := range strings.SplitSeq(value, ",") {
		if column = strings.TrimSpace(column); column != "" {
			columns = append(columns, column)
		}
	}
	return columns
}

// validColumn reports whether a report column is supported.
func validColumn(column string) bool {
	switch column {
	case "title", "path", "status", "owner", "updated", "author", "tags", "views":
		return true
	}
	key, ok := strings.CutPrefix(column, "property:")
	return ok && strings.TrimSpace(key) != ""
}
