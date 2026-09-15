package main

import (
	"io"
	"strings"
	"unicode"

	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const preserveOperatorsPolicy = "preserve-programming-operators"

func main() {}

func init() { sdk.RegisterModule("typographer", transform) }

func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "typographer" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported typographer render request"}
	}

	output, err := typographHTML(request.Source, sdk.RenderPolicyEnabled(request.Features, preserveOperatorsPolicy))
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}

type quoteState struct {
	previous rune
	single   int
	double   int
}

func typographHTML(source string, preserveOperators bool) (string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	state := &quoteState{}
	skipDepth := 0
	var output strings.Builder

	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if err := tokenizer.Err(); err != io.EOF {
				return "", err
			}
			return output.String(), nil
		}

		raw := string(tokenizer.Raw())
		switch tokenType {
		case xhtml.TextToken:
			if skipDepth > 0 {
				output.WriteString(raw)
				continue
			}

			token := tokenizer.Token()
			transformed := typographText(token.Data, state, preserveOperators)
			if transformed == token.Data {
				output.WriteString(raw)
				continue
			}
			writeHTMLText(&output, transformed)

		case xhtml.StartTagToken:
			token := tokenizer.Token()
			output.WriteString(raw)
			if skipTypography(token.DataAtom) {
				skipDepth++
				state.previous = ' '
				continue
			}
			if skipDepth == 0 && blockBoundary(token.DataAtom) {
				*state = quoteState{}
			}

		case xhtml.EndTagToken:
			token := tokenizer.Token()
			output.WriteString(raw)
			if skipTypography(token.DataAtom) && skipDepth > 0 {
				skipDepth--
				state.previous = ' '
				continue
			}
			if skipDepth == 0 && blockBoundary(token.DataAtom) {
				*state = quoteState{}
			}

		default:
			output.WriteString(raw)
		}
	}
}

func writeHTMLText(output *strings.Builder, text string) {
	for _, value := range text {
		switch value {
		case '&':
			output.WriteString("&amp;")
		case '<':
			output.WriteString("&lt;")
		case '>':
			output.WriteString("&gt;")
		default:
			output.WriteRune(value)
		}
	}
}

func skipTypography(tag atom.Atom) bool {
	switch tag {
	case atom.Code, atom.Pre, atom.Kbd, atom.Samp, atom.Script, atom.Style:
		return true
	default:
		return false
	}
}

func blockBoundary(tag atom.Atom) bool {
	switch tag {
	case atom.P, atom.Li, atom.Blockquote, atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6, atom.Td, atom.Th:
		return true
	default:
		return false
	}
}

func typographText(text string, state *quoteState, preserveOperators bool) string {
	runes := []rune(text)
	var output strings.Builder
	for index := 0; index < len(runes); {
		if hasRunes(runes, index, '.', '.', '.') {
			writeRune(&output, state, '…')
			index += 3
			continue
		}
		if !preserveOperators && hasRunes(runes, index, '-', '-', '-') {
			writeRune(&output, state, '—')
			index += 3
			continue
		}
		if !preserveOperators && hasRunes(runes, index, '-', '-') {
			writeRune(&output, state, '–')
			index += 2
			continue
		}
		if !preserveOperators && hasRunes(runes, index, '<', '<') {
			writeRune(&output, state, '«')
			index += 2
			continue
		}
		if !preserveOperators && hasRunes(runes, index, '>', '>') {
			writeRune(&output, state, '»')
			index += 2
			continue
		}

		current := runes[index]
		next := rune(0)
		if index+1 < len(runes) {
			next = runes[index+1]
		}
		switch current {
		case '\'':
			writeRune(&output, state, smartSingleQuote(state.previous, next, state))
		case '"':
			writeRune(&output, state, smartDoubleQuote(state.previous, next, state))
		default:
			writeRune(&output, state, current)
		}
		index++
	}
	return output.String()
}

func smartSingleQuote(previous, next rune, state *quoteState) rune {
	if isAlphaNumeric(previous) && unicode.IsLetter(next) {
		return '’'
	}
	if openingContext(previous) && (unicode.IsDigit(next) || isApostrophePrefix(next)) {
		return '’'
	}
	if openingQuoteContext(previous, next, state.single) {
		state.single++
		return '‘'
	}
	if state.single > 0 {
		state.single--
	}
	return '’'
}

func smartDoubleQuote(previous, next rune, state *quoteState) rune {
	if openingQuoteContext(previous, next, state.double) {
		state.double++
		return '“'
	}
	if state.double > 0 {
		state.double--
	}
	return '”'
}

func openingQuoteContext(previous, next rune, depth int) bool {
	if !openingContext(previous) {
		return false
	}
	if next == 0 {
		return depth == 0
	}
	return !closingContext(next)
}

func openingContext(value rune) bool {
	return value == 0 || unicode.IsSpace(value) || unicode.IsPunct(value)
}

func closingContext(value rune) bool {
	return value == 0 || unicode.IsSpace(value)
}

func isAlphaNumeric(value rune) bool {
	return unicode.IsLetter(value) || unicode.IsDigit(value)
}

func isApostrophePrefix(value rune) bool {
	switch unicode.ToLower(value) {
	case 't', 'e', 'n', 'l':
		return true
	default:
		return false
	}
}

func writeRune(output *strings.Builder, state *quoteState, value rune) {
	output.WriteRune(value)
	state.previous = value
}

func hasRunes(source []rune, offset int, values ...rune) bool {
	if offset+len(values) > len(source) {
		return false
	}
	for index, value := range values {
		if source[offset+index] != value {
			return false
		}
	}
	return true
}
