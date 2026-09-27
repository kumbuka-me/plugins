// Package macroargs parses the compact key=value argument syntax shared by first-party macros.
package macroargs

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// parser tracks one byte-oriented macro argument scan.
type parser struct {
	// value is the complete argument string being parsed.
	value string
	// index is the next unread byte in value.
	index int
}

// ParseUnique parses unique lowercase key=value arguments with quoted or bare values.
func ParseUnique(value string) (map[string]string, bool) {
	parser := parser{value: value}
	arguments := make(map[string]string)
	for {
		name, value, done, ok := parser.next()
		if !ok {
			return nil, false
		}
		if done {
			return arguments, true
		}
		if _, exists := arguments[name]; exists {
			return nil, false
		}
		arguments[name] = value
	}
}

// next parses one name=value argument or reports the end of input.
func (p *parser) next() (name, value string, done, ok bool) {
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
func (p *parser) skipSpace() {
	for p.index < len(p.value) && (p.value[p.index] == ' ' || p.value[p.index] == '\t') {
		p.index++
	}
}

// readName consumes one lowercase option name.
func (p *parser) readName() string {
	start := p.index
	for p.index < len(p.value) {
		character := p.value[p.index]
		if character >= 'a' && character <= 'z' || character == '_' {
			p.index++
			continue
		}
		break
	}
	return p.value[start:p.index]
}

// readValue consumes one quoted or unquoted option value.
func (p *parser) readValue() (string, bool) {
	if p.index >= len(p.value) {
		return "", false
	}
	if p.value[p.index] == '"' {
		return p.readQuotedValue()
	}
	return p.readBareValue()
}

// readBareValue consumes a value up to the next horizontal whitespace.
func (p *parser) readBareValue() (string, bool) {
	start := p.index
	for p.index < len(p.value) && p.value[p.index] != ' ' && p.value[p.index] != '\t' {
		p.index++
	}
	return p.value[start:p.index], p.index > start
}

// readQuotedValue consumes and unquotes a double-quoted option value.
func (p *parser) readQuotedValue() (string, bool) {
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

// NextQuoted consumes one name="value" attribute and returns the remaining body.
func NextQuoted(body string) (name, value, remaining string, ok bool) {
	body = strings.TrimSpace(body)
	name, rest, found := strings.Cut(body, "=")
	if !found {
		return "", "", "", false
	}

	name = strings.TrimSpace(name)
	rest = strings.TrimSpace(rest)
	if name == "" || len(rest) < 2 || rest[0] != '"' {
		return "", "", "", false
	}

	end := closingQuote(rest)
	if end < 0 {
		return "", "", "", false
	}
	value, err := strconv.Unquote(rest[:end+1])
	if err != nil || !utf8.ValidString(value) {
		return "", "", "", false
	}

	remaining = rest[end+1:]
	if remaining != "" && remaining[0] != ' ' && remaining[0] != '\t' {
		return "", "", "", false
	}
	return name, value, remaining, true
}

// closingQuote returns the closing quote index for one escaped quoted value.
func closingQuote(value string) int {
	for index := 1; index < len(value); index++ {
		if value[index] == '\\' {
			index++
			continue
		}
		if value[index] == '"' {
			return index
		}
	}
	return -1
}
