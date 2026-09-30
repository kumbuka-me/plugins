package main

import (
	"io"
	"sort"
	"strconv"
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
	// htmlSpan bounds the complete table in the source HTML.
	htmlSpan
	// directives contains markers applied in document order.
	directives []tableStyle
	// hasRows records whether this table contains at least one row.
	hasRows bool
}

// htmlReplacement describes one bounded source replacement.
type htmlReplacement struct {
	// htmlSpan bounds the source bytes replaced by text.
	htmlSpan
	// text replaces the range; an empty value removes it.
	text string
}

// processRenderedTables applies table directives and optional browser wrappers without constructing a DOM for the complete rendered page. Individual table fragments use a streaming rewrite; malformed fragments containing nested legacy markers retain the DOM fallback for compatibility.
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
		replacement, err := transformTableFragment(source[table.start:table.end], table.directives, options, wrap, table.hasRows)
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

// scanRenderedTables records table and directive byte ranges using the streaming HTML tokenizer. Rendered directive paragraphs are consumed only when they immediately follow a table; legacy marker divs remain supported.
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

			if token.Data == "tr" && len(tableStack) != 0 {
				tables[tableStack[len(tableStack)-1]].hasRows = true
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

// transformTableFragment rewrites the common table path without allocating a DOM.
// Legacy markers inside a table are sufficiently unusual that they retain the
// DOM implementation, keeping malformed/hand-authored compatibility off the hot path.
func transformTableFragment(source string, directives []tableStyle, options tableOptions, wrap, hasRows bool) (string, error) {
	if strings.Contains(source, "kumbuka-table-style-marker") {
		return transformTableFragmentDOM(source, directives, options, wrap)
	}
	return transformTableFragmentStreaming(source, directives, options, wrap, hasRows)
}

// transformTableFragmentStreaming applies trusted table changes while copying all unmodified HTML tokens directly from the source.
func transformTableFragmentStreaming(source string, directives []tableStyle, options tableOptions, wrap, hasRows bool) (string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var output strings.Builder
	output.Grow(len(source) + 160)

	tableDepth := 0
	theadDepth := 0
	rowIndex := -1
	columnIndex := -1
	firstRowInThead := false
	rowOpen := false

	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return "", tokenizer.Err()
		}

		raw := tokenizer.Raw()
		switch tokenType {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			selfClosing := tokenType == xhtml.SelfClosingTagToken

			switch token.Data {
			case "table":
				if tableDepth == 0 {
					applyTableToken(&token, directives, options, hasRows)
					if wrap {
						writeTableWrapperStart(&output, tokenHasClass(token, "kumbuka-table-sortable") || tokenHasClass(token, "kumbuka-table-filterable"))
					}
					output.WriteString(token.String())
				} else {
					output.Write(raw)
				}
				if !selfClosing {
					tableDepth++
				} else if tableDepth == 0 && wrap {
					writeTableWrapperEnd(&output)
				}
				continue

			case "thead":
				if tableDepth == 1 && !selfClosing {
					theadDepth++
				}

			case "tr":
				if tableDepth == 1 {
					rowIndex++
					columnIndex = -1
					rowOpen = !selfClosing
					if rowIndex == 0 {
						firstRowInThead = theadDepth > 0
					}
					applyTableRowToken(&token, directives, options, rowIndex)
					output.WriteString(token.String())
					continue
				}

			case "th", "td":
				if tableDepth == 1 && rowOpen {
					columnIndex++
					applyTableCellToken(&token, directives, options, rowIndex, columnIndex, firstRowInThead)
					output.WriteString(token.String())
					continue
				}
			}

			output.Write(raw)

		case xhtml.EndTagToken:
			token := tokenizer.Token()
			switch token.Data {
			case "table":
				if tableDepth == 1 {
					output.Write(raw)
					tableDepth--
					if wrap {
						writeTableWrapperEnd(&output)
					}
					continue
				}
				if tableDepth > 1 {
					tableDepth--
				}
			case "thead":
				if tableDepth == 1 && theadDepth > 0 {
					theadDepth--
				}
			case "tr":
				if tableDepth == 1 {
					rowOpen = false
				}
			}
			output.Write(raw)

		default:
			output.Write(raw)
		}
	}

	return output.String(), nil
}

// applyTableToken applies table-level interaction, style, and dimension attributes.
func applyTableToken(token *xhtml.Token, directives []tableStyle, options tableOptions, hasRows bool) {
	for _, directive := range directives {
		if options.TableSorting && directive.sortable {
			addTokenClass(token, "kumbuka-table-sortable")
		}
		if options.TableFiltering && directive.filterable {
			addTokenClass(token, "kumbuka-table-filterable")
		}
		if !options.TableStyles || !directive.hasStyling() || !hasRows {
			continue
		}
		if width, fixed := fixedTableWidth(directive.widths); fixed {
			token.Attr = append(token.Attr, xhtml.Attribute{Key: "style", Val: "table-layout:fixed;width:" + strconv.Itoa(width) + "px"})
		}
		addTokenClass(token, "kumbuka-table-styled")
	}
}

// applyTableRowToken applies configured row heights in document order.
func applyTableRowToken(token *xhtml.Token, directives []tableStyle, options tableOptions, rowIndex int) {
	if !options.TableStyles {
		return
	}
	for _, directive := range directives {
		if rowIndex < len(directive.heights) && directive.heights[rowIndex] > 0 {
			token.Attr = append(token.Attr, xhtml.Attribute{Key: "style", Val: "height:" + strconv.Itoa(directive.heights[rowIndex]) + "px"})
		}
	}
}

// applyTableCellToken applies configured widths and the final tone for one cell.
func applyTableCellToken(token *xhtml.Token, directives []tableStyle, options tableOptions, rowIndex, columnIndex int, firstRowInThead bool) {
	if !options.TableStyles {
		return
	}

	tone := ""
	for _, directive := range directives {
		if columnIndex < len(directive.widths) && directive.widths[columnIndex] > 0 {
			token.Attr = append(token.Attr, xhtml.Attribute{Key: "style", Val: "width:" + strconv.Itoa(directive.widths[columnIndex]) + "px"})
		}
		if rowIndex == 0 && directive.header != "" {
			tone = directive.header
		}
		if value, ok := directive.columns[columnIndex+1]; ok {
			tone = value
		}

		bodyRow := rowIndex + 1
		if firstRowInThead {
			if rowIndex == 0 {
				bodyRow = 0
			} else {
				bodyRow = rowIndex
			}
		}
		if bodyRow > 0 {
			if value, ok := directive.rows[bodyRow]; ok {
				tone = value
			}
			if value, ok := directive.cells[[2]int{bodyRow, columnIndex + 1}]; ok {
				tone = value
			}
		}
	}
	if tone != "" {
		setTokenTone(token, tone)
	}
}

// writeTableWrapperStart emits the browser-module fallback wrapper without constructing DOM nodes.
func writeTableWrapperStart(output *strings.Builder, interactive bool) {
	output.WriteString(`<div class="kumbuka-plugin-block" data-kumbuka-plugin="me.kumbuka.tables" data-kumbuka-input="html"`)
	if interactive {
		output.WriteString(` data-kumbuka-module="interactive"`)
	}
	output.WriteString(`><div data-kumbuka-fallback="">`)
}

// writeTableWrapperEnd closes the browser-module fallback wrapper.
func writeTableWrapperEnd(output *strings.Builder) { output.WriteString(`</div></div>`) }

// addTokenClass adds one class to a parsed start tag when absent.
func addTokenClass(token *xhtml.Token, className string) {
	classes := strings.Fields(tokenAttribute(*token, "class"))
	for _, existing := range classes {
		if existing == className {
			return
		}
	}
	classes = append(classes, className)
	setTokenAttribute(token, "class", strings.Join(classes, " "))
}

// setTokenTone replaces prior trusted tone classes while retaining unrelated classes.
func setTokenTone(token *xhtml.Token, tone string) {
	const prefix = "table-tone-"
	classes := strings.Fields(tokenAttribute(*token, "class"))
	filtered := classes[:0]
	for _, className := range classes {
		if !strings.HasPrefix(className, prefix) {
			filtered = append(filtered, className)
		}
	}
	filtered = append(filtered, prefix+tone)
	setTokenAttribute(token, "class", strings.Join(filtered, " "))
}

// setTokenAttribute replaces or appends one parsed token attribute.
func setTokenAttribute(token *xhtml.Token, key, value string) {
	for index := range token.Attr {
		if token.Attr[index].Key == key {
			token.Attr[index].Val = value
			return
		}
	}
	token.Attr = append(token.Attr, xhtml.Attribute{Key: key, Val: value})
}

// transformTableFragmentDOM parses and mutates one table for compatibility with legacy markers nested inside the table fragment.
func transformTableFragmentDOM(source string, directives []tableStyle, options tableOptions, wrap bool) (string, error) {
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
