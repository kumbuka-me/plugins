package main

import (
	stdhtml "html"
	"slices"
	"strconv"
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	pluginmarkdown "github.com/kumbuka-me/sdk/markdown"
	xhtml "golang.org/x/net/html"
)

// tableOptions contains the request-scoped table feature switches.
type tableOptions struct {
	// Tables enables table processing for the request.
	Tables bool
	// TableStyles enables trusted tone directives.
	TableStyles bool
	// TableSorting enables client-side sortable table markers.
	TableSorting bool
	// TableFiltering enables client-side filterable table markers.
	TableFiltering bool
}

// tableStyle describes trusted presentation classes applied to one rendered table.
type tableStyle struct {
	widths  []int
	heights []int
	// header is the optional header-row tone.
	header string
	// rows maps one-based body rows to tones.
	rows map[int]string
	// columns maps one-based columns to tones.
	columns map[int]string
	// cells maps one-based body-row and column positions to tones.
	cells map[[2]int]string
	// sortable marks the table for client-side column sorting.
	sortable bool
	// filterable marks the table for client-side row filtering.
	filterable bool
}

// tableDirectivesEnabled reports whether any table directive feature can be rendered.
func tableDirectivesEnabled(options tableOptions) bool {
	return options.Tables &&
		(options.TableStyles ||
			options.TableSorting ||
			options.TableFiltering)
}

// preprocessTableDirectives replaces enabled table directives with trusted markers consumed after rendering.
func preprocessTableDirectives(
	source string,
	options tableOptions,
) string {
	lines := strings.Split(source, "\n")
	out := make([]string, 0, len(lines))
	fence := ""

	for index, line := range lines {
		marker := pluginmarkdown.Fence(line)

		if fence != "" {
			out = append(out, line)

			if pluginmarkdown.Closes(line, fence) {
				fence = ""
			}

			continue
		}

		if marker != "" {
			fence = marker
			out = append(out, line)
			continue
		}

		trimmed := strings.TrimSpace(line)

		if previousTableLine(lines, index) {
			directive, ok := parseTableDirective(trimmed)

			if ok &&
				tableDirectiveActive(directive, options) {
				out = append(
					out,
					`<div class="kumbuka-table-style-marker" data-table-style="`+
						stdhtml.EscapeString(trimmed)+
						`"></div>`,
				)

				continue
			}
		}

		out = append(out, line)
	}

	return strings.Join(out, "\n")
}

// previousTableLine reports whether a directive immediately follows a Markdown table row.
func previousTableLine(
	lines []string,
	index int,
) bool {
	for _, line := range slices.Backward(lines[:index]) {
		if strings.TrimSpace(line) == "" {
			continue
		}

		return strings.Contains(line, "|")
	}

	return false
}

// parseTableDirective parses trusted table colors and optional browser interactions.
func parseTableDirective(line string) (tableStyle, bool) {
	body, ok := tableDirectiveBody(line)
	if !ok {
		return tableStyle{}, false
	}

	style := newTableStyle()
	for token := range strings.FieldsSeq(body) {
		if !style.applyToken(token) {
			return tableStyle{}, false
		}
	}
	return style, true
}

// tableDirectiveBody extracts a non-empty table directive body.
func tableDirectiveBody(line string) (string, bool) {
	body, ok := strings.CutPrefix(line, "{table ")
	if !ok {
		return "", false
	}
	body, ok = strings.CutSuffix(body, "}")
	body = strings.TrimSpace(body)
	return body, ok && body != ""
}

// newTableStyle initializes the indexed tone maps used by one table directive.
func newTableStyle() tableStyle {
	return tableStyle{
		rows:    map[int]string{},
		columns: map[int]string{},
		cells:   map[[2]int]string{},
	}
}

// applyToken adds one directive token to the parsed style.
func (s *tableStyle) applyToken(token string) bool {
	switch token {
	case "sortable":
		s.sortable = true
		return true
	case "filterable":
		s.filterable = true
		return true
	}

	key, tone, ok := strings.Cut(token, "=")
	if ok && (key == "widths" || key == "heights") {
		values := []int{}
		for _, raw := range strings.Split(tone, ",") {
			value, err := strconv.Atoi(raw)
			if err != nil || value < 0 || value > 4000 {
				return false
			}
			values = append(values, value)
		}
		if key == "widths" {
			s.widths = values
		} else {
			s.heights = values
		}
		return true
	}
	if !ok || !tableTone(tone) {
		return false
	}
	return s.setTone(key, tone)
}

// setTone assigns one trusted tone to a header, row, column, or cell target.
func (s *tableStyle) setTone(key, tone string) bool {
	if key == "header" {
		s.header = tone
		return true
	}

	kind, target, ok := strings.Cut(key, ":")
	if !ok {
		return false
	}

	switch kind {
	case "row":
		row, ok := parsePositiveInt(target)
		if ok {
			s.rows[row] = tone
		}
		return ok
	case "col", "column":
		column, ok := parsePositiveInt(target)
		if ok {
			s.columns[column] = tone
		}
		return ok
	case "cell":
		position, ok := parseCellPosition(target)
		if ok {
			s.cells[position] = tone
		}
		return ok
	default:
		return false
	}
}

// parseCellPosition parses a one-based row,column table coordinate.
func parseCellPosition(raw string) ([2]int, bool) {
	rowValue, columnValue, ok := strings.Cut(raw, ",")
	if !ok {
		return [2]int{}, false
	}
	row, rowOK := parsePositiveInt(rowValue)
	column, columnOK := parsePositiveInt(columnValue)
	return [2]int{row, column}, rowOK && columnOK
}

// parsePositiveInt parses a one-based table row or column index.
func parsePositiveInt(
	raw string,
) (value int, ok bool) {
	value, err := strconv.Atoi(raw)

	if err != nil || value < 1 {
		return 0, false
	}

	return value, true
}

// tableDirectiveActive reports whether a parsed directive contains any currently enabled behavior.
func tableDirectiveActive(
	directive tableStyle,
	options tableOptions,
) bool {
	return (directive.hasColors() && options.TableStyles) ||
		(directive.sortable && options.TableSorting) ||
		(directive.filterable && options.TableFiltering)
}

// tableTone reports whether a table color maps to a trusted theme-aware palette class.
func tableTone(value string) bool {
	switch value {
	case "accent",
		"accent-soft",
		"info",
		"success",
		"warning",
		"danger",
		"neutral",
		"gray",
		"blue",
		"purple",
		"green",
		"yellow",
		"orange",
		"red":
		return true
	default:
		return false
	}
}

// tableDirectiveWalker tracks the nearest table while applying rendered table markers.
type tableDirectiveWalker struct {
	// options contains the request-scoped table feature switches.
	options tableOptions
	// markers contains directive marker nodes removed after processing.
	markers []*xhtml.Node
	// lastTable is the nearest preceding rendered table in document order.
	lastTable *xhtml.Node
}

// applyTableDirectiveMarkers applies trusted directives to the nearest preceding rendered table.
func applyTableDirectiveMarkers(
	rendered string,
	options tableOptions,
) (string, error) {
	root, err := htmlutil.ParseFragment(rendered)
	if err != nil {
		return "", err
	}

	walker := tableDirectiveWalker{options: options}
	walker.walk(root)

	for _, marker := range walker.markers {
		if marker.Parent != nil {
			marker.Parent.RemoveChild(marker)
		}
	}

	return htmlutil.RenderChildren(root)
}

// walk applies one table marker in document order and records it for removal.
func (w *tableDirectiveWalker) walk(
	node *xhtml.Node,
) {
	if node.Type == xhtml.ElementNode {
		if node.Data == "table" {
			w.lastTable = node
		}

		if node.Data == "div" &&
			strings.Contains(
				" "+htmlutil.Attribute(node, "class")+" ",
				" kumbuka-table-style-marker ",
			) {
			directive, ok := parseTableDirective(
				htmlutil.Attribute(node, "data-table-style"),
			)

			if ok && w.lastTable != nil {
				applyTableDirective(
					w.lastTable,
					directive,
					w.options,
				)
			}

			w.markers = append(w.markers, node)

			return
		}
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		w.walk(child)
	}
}

// applyTableDirective applies enabled colors and interaction classes to one rendered table.
func applyTableDirective(table *xhtml.Node, directive tableStyle, options tableOptions) {
	if options.TableSorting && directive.sortable {
		htmlutil.AddClass(table, "kumbuka-table-sortable")
	}
	if options.TableFiltering && directive.filterable {
		htmlutil.AddClass(table, "kumbuka-table-filterable")
	}
	if !options.TableStyles || !directive.hasColors() {
		return
	}

	rows := tableRows(table)
	if len(rows) == 0 {
		return
	}

	if len(directive.widths) > 0 {
		total := 0
		for _, width := range directive.widths {
			if width == 0 {
				total += 100
			} else {
				total += width
			}
		}
		table.Attr = append(table.Attr, xhtml.Attribute{Key: "style", Val: "table-layout:fixed;width:" + strconv.Itoa(total) + "px"})
	}
	for index, row := range rows {
		if index < len(directive.heights) && directive.heights[index] > 0 {
			row.Attr = append(row.Attr, xhtml.Attribute{Key: "style", Val: "height:" + strconv.Itoa(directive.heights[index]) + "px"})
		}
		for column, cell := range rowCells(row) {
			if column < len(directive.widths) && directive.widths[column] > 0 {
				cell.Attr = append(cell.Attr, xhtml.Attribute{Key: "style", Val: "width:" + strconv.Itoa(directive.widths[column]) + "px"})
			}
		}
	}
	htmlutil.AddClass(table, "kumbuka-table-styled")
	applyHeaderTone(rows, directive.header)
	applyColumnTones(rows, directive.columns)
	bodyRows := tableBodyRows(rows)
	applyRowTones(bodyRows, directive.rows)
	applyCellTones(bodyRows, directive.cells)
}

// hasColors reports whether a directive contains any style tone assignments.
func (s tableStyle) hasColors() bool {
	return len(s.widths) != 0 || len(s.heights) != 0 || s.header != "" || len(s.rows) != 0 || len(s.columns) != 0 || len(s.cells) != 0
}

// applyHeaderTone colors every cell in the rendered header row when configured.
func applyHeaderTone(rows []*xhtml.Node, tone string) {
	if tone == "" {
		return
	}
	for _, cell := range rowCells(rows[0]) {
		setTableTone(cell, tone)
	}
}

// applyColumnTones colors configured one-based columns across all rendered rows.
func applyColumnTones(rows []*xhtml.Node, tones map[int]string) {
	for column, tone := range tones {
		for _, row := range rows {
			cells := rowCells(row)
			if column <= len(cells) {
				setTableTone(cells[column-1], tone)
			}
		}
	}
}

// tableBodyRows removes the header row when the first row belongs to a thead section.
func tableBodyRows(rows []*xhtml.Node) []*xhtml.Node {
	if len(rows) != 0 && hasAncestorSection(rows[0], "thead") {
		return rows[1:]
	}
	return rows
}

// applyRowTones colors configured one-based body rows.
func applyRowTones(rows []*xhtml.Node, tones map[int]string) {
	for row, tone := range tones {
		if row > len(rows) {
			continue
		}
		for _, cell := range rowCells(rows[row-1]) {
			setTableTone(cell, tone)
		}
	}
}

// applyCellTones colors configured one-based body-row and column coordinates.
func applyCellTones(rows []*xhtml.Node, tones map[[2]int]string) {
	for position, tone := range tones {
		row, column := position[0], position[1]
		if row > len(rows) {
			continue
		}
		cells := rowCells(rows[row-1])
		if column <= len(cells) {
			setTableTone(cells[column-1], tone)
		}
	}
}

// tableRows returns table rows in document order.
func tableRows(
	table *xhtml.Node,
) []*xhtml.Node {
	var rows []*xhtml.Node

	appendTableRows(table, &rows)

	return rows
}

// appendTableRows collects row elements without descending into a row once found.
func appendTableRows(
	node *xhtml.Node,
	rows *[]*xhtml.Node,
) {
	if node.Type == xhtml.ElementNode &&
		node.Data == "tr" {
		*rows = append(*rows, node)
		return
	}

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		appendTableRows(child, rows)
	}
}

// rowCells returns direct th and td children for one rendered row.
func rowCells(
	row *xhtml.Node,
) []*xhtml.Node {
	var cells []*xhtml.Node

	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode &&
			(child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}

	return cells
}

// hasAncestorSection reports whether a row sits below the named table section.
func hasAncestorSection(
	node *xhtml.Node,
	section string,
) bool {
	for current := node.Parent; current != nil; current = current.Parent {
		if current.Type == xhtml.ElementNode &&
			current.Data == section {
			return true
		}

		if current.Type == xhtml.ElementNode &&
			current.Data == "table" {
			return false
		}
	}

	return false
}

// addHTMLClass adds a class to one rendered HTML element when it is not already present.
// setTableTone replaces a previously applied tone with the requested theme-aware class.
func setTableTone(
	node *xhtml.Node,
	tone string,
) {
	const prefix = "table-tone-"

	classes := strings.Fields(
		htmlutil.Attribute(node, "class"),
	)

	classes = slices.DeleteFunc(
		classes,
		func(className string) bool {
			return strings.HasPrefix(
				className,
				prefix,
			)
		},
	)

	classes = append(classes, prefix+tone)

	htmlutil.SetAttribute(
		node,
		"class",
		strings.Join(classes, " "),
	)
}

// setHTMLAttribute sets or appends one HTML node attribute.
// htmlAttribute returns one HTML node attribute by key.
