package main

import (
	"io"
	"sort"
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	xhtml "golang.org/x/net/html"
)

// htmlSpan identifies one half-open byte range in rendered HTML.
type htmlSpan struct {
	// start is the first byte in the range.
	start int
	// end is the first byte after the range.
	end int
}

// renderedTableSpan records one table and the directives associated with it.
type renderedTableSpan struct {
	htmlSpan
	// directives contains markers applied in document order.
	directives []tableStyle
}

// htmlReplacement describes one bounded source replacement.
type htmlReplacement struct {
	htmlSpan
	// text replaces the range; an empty value removes it.
	text string
}

// processRenderedTables applies table directives and optional browser wrappers
// without constructing a DOM for the complete rendered page. Only individual
// table fragments are parsed; all unrelated HTML is copied directly.
func processRenderedTables(source string, options tableOptions, wrap bool) (string, error) {
	tables, markers, err := scanRenderedTables(source, options)
	if err != nil {
		return "", err
	}
	if len(tables) == 0 && len(markers) == 0 {
		return source, nil
	}

	replacements := make([]htmlReplacement, 0, len(tables)+len(markers))
	for _, table := range tables {
		if table.end <= table.start {
			continue
		}
		replacement, err := transformTableFragment(source[table.start:table.end], table.directives, options, wrap)
		if err != nil {
			return "", err
		}
		replacements = append(replacements, htmlReplacement{htmlSpan: table.htmlSpan, text: replacement})
	}
	for _, marker := range markers {
		if marker.end > marker.start {
			replacements = append(replacements, htmlReplacement{htmlSpan: marker})
		}
	}

	sort.Slice(replacements, func(i, j int) bool {
		if replacements[i].start == replacements[j].start {
			return replacements[i].end > replacements[j].end
		}
		return replacements[i].start < replacements[j].start
	})

	var output strings.Builder
	output.Grow(len(source))
	cursor := 0
	for _, replacement := range replacements {
		if replacement.start < cursor {
			continue
		}
		output.WriteString(source[cursor:replacement.start])
		output.WriteString(replacement.text)
		cursor = replacement.end
	}
	output.WriteString(source[cursor:])
	return output.String(), nil
}

// scanRenderedTables records table and directive byte ranges using the streaming
// HTML tokenizer. Rendered directive paragraphs are consumed only when they
// immediately follow a table; legacy marker divs remain supported.
func scanRenderedTables(source string, options tableOptions) ([]renderedTableSpan, []htmlSpan, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var tables []renderedTableSpan
	var tableStack []int
	var markers []htmlSpan
	lastTable := -1
	offset := 0
	activeMarker := -1
	markerDepth := 0
	directiveStart := -1
	directiveTable := -1
	directiveNested := false
	var directiveText strings.Builder

	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return nil, nil, tokenizer.Err()
		}

		raw := tokenizer.Raw()
		start := offset
		offset += len(raw)

		switch tokenType {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			selfClosing := tokenType == xhtml.SelfClosingTagToken

			if activeMarker >= 0 && token.Data == "div" && !selfClosing {
				markerDepth++
			}

			if directiveStart >= 0 {
				directiveNested = true
			}

			if token.Data == "table" {
				tables = append(tables, renderedTableSpan{htmlSpan: htmlSpan{start: start}})
				lastTable = len(tables) - 1
				if selfClosing {
					tables[lastTable].end = offset
				} else {
					tableStack = append(tableStack, lastTable)
				}
			}

			if activeMarker < 0 && token.Data == "div" && tokenHasClass(token, "kumbuka-table-style-marker") {
				if directive, ok := parseTableDirective(tokenAttribute(token, "data-table-style")); ok && lastTable >= 0 {
					tables[lastTable].directives = append(tables[lastTable].directives, directive)
				}
				markers = append(markers, htmlSpan{start: start})
				activeMarker = len(markers) - 1
				markerDepth = 1
				if selfClosing {
					markers[activeMarker].end = offset
					activeMarker = -1
					markerDepth = 0
				}
			}

			if directiveStart < 0 && !selfClosing && token.Data == "p" && tableDirectivesEnabled(options) && lastTable >= 0 && tables[lastTable].end > tables[lastTable].start && strings.TrimSpace(source[tables[lastTable].end:start]) == "" {
				directiveStart = start
				directiveTable = lastTable
				directiveNested = false
				directiveText.Reset()
			}

		case xhtml.TextToken:
			if directiveStart >= 0 && !directiveNested {
				directiveText.WriteString(tokenizer.Token().Data)
			}

		case xhtml.EndTagToken:
			token := tokenizer.Token()
			if token.Data == "table" && len(tableStack) != 0 {
				index := tableStack[len(tableStack)-1]
				tableStack = tableStack[:len(tableStack)-1]
				tables[index].end = offset
			}
			if activeMarker >= 0 && token.Data == "div" {
				markerDepth--
				if markerDepth == 0 {
					markers[activeMarker].end = offset
					activeMarker = -1
				}
			}
			if directiveStart >= 0 && token.Data == "p" {
				if !directiveNested {
					if directive, ok := parseTableDirective(strings.TrimSpace(directiveText.String())); ok && tableDirectiveActive(directive, options) {
						tables[directiveTable].directives = append(tables[directiveTable].directives, directive)
						markers = append(markers, htmlSpan{start: directiveStart, end: offset})
					}
				}
				directiveStart = -1
				directiveTable = -1
				directiveNested = false
				directiveText.Reset()
			}
		}
	}

	return tables, markers, nil
}

// transformTableFragment parses and mutates one table instead of the complete
// page. Nested marker nodes are removed here so overlapping marker ranges stay
// safe if malformed or hand-authored HTML places one inside a table.
func transformTableFragment(source string, directives []tableStyle, options tableOptions, wrap bool) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}

	table := firstTable(root)
	if table == nil {
		return source, nil
	}
	removeTableDirectiveMarkers(root)
	for _, directive := range directives {
		applyTableDirective(table, directive, options)
	}
	if wrap {
		wrapTable(table)
	}
	return htmlutil.RenderChildren(root)
}

// firstTable returns the first table below node in document order.
func firstTable(node *xhtml.Node) *xhtml.Node {
	if node.Type == xhtml.ElementNode && node.Data == "table" {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if table := firstTable(child); table != nil {
			return table
		}
	}
	return nil
}

// removeTableDirectiveMarkers removes directive markers contained by one table fragment.
func removeTableDirectiveMarkers(node *xhtml.Node) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if child.Type == xhtml.ElementNode && child.Data == "div" && strings.Contains(" "+htmlutil.Attribute(child, "class")+" ", " kumbuka-table-style-marker ") {
			node.RemoveChild(child)
		} else {
			removeTableDirectiveMarkers(child)
		}
		child = next
	}
}

// tokenAttribute returns one parsed token attribute by key.
func tokenAttribute(token xhtml.Token, key string) string {
	for _, attribute := range token.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

// tokenHasClass reports whether one parsed token contains a class name.
func tokenHasClass(token xhtml.Token, className string) bool {
	for _, class := range strings.Fields(tokenAttribute(token, "class")) {
		if class == className {
			return true
		}
	}
	return false
}
