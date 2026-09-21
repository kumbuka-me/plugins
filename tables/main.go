package main

import (
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Tables preprocess and postprocess modules.
func init() {
	sdk.RegisterModule("directives", transform)
	sdk.RegisterModule("presentation", transform)
}

// transform applies request-scoped table directives and presentation wrapping for the active stage.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	options := tableOptionsFromFeatures(request.Features)

	switch request.Stage {
	case "preprocess":
		output := request.Source
		if tableDirectivesEnabled(options) {
			output = preprocessTableDirectives(output, options)
		}
		return sdk.Text(output)
	case "postprocess":
		output, err := postprocessTables(request.Source, options)
		if err != nil {
			return sdk.Failure(err)
		}
		return sdk.Text(output)
	default:
		return sdk.RenderResult{Error: "unsupported stage"}
	}
}

// tableOptionsFromFeatures resolves the enabled table feature switches for one request.
func tableOptionsFromFeatures(features map[string]bool) tableOptions {
	enabled := func(name string) bool {
		value, exists := features["me.kumbuka.tables."+name]
		return !exists || value
	}
	return tableOptions{
		Tables:         enabled("tables"),
		TableStyles:    enabled("styles"),
		TableSorting:   enabled("sorting"),
		TableFiltering: enabled("filtering"),
	}
}

// postprocessTables applies directives before wrapping rendered tables for browser modules.
func postprocessTables(source string, options tableOptions) (string, error) {
	output := source
	var err error
	if tableDirectivesEnabled(options) {
		output, err = applyTableDirectiveMarkers(output, options)
		if err != nil {
			return "", err
		}
	}
	return markTables(output)
}

// markTables leaves semantic HTML as the accessible, script-free fallback.
func markTables(source string) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}

	for _, table := range collectTables(root) {
		wrapTable(table)
	}
	return htmlutil.RenderChildren(root)
}

// collectTables returns rendered tables in document order without descending into a table twice.
func collectTables(root *xhtml.Node) []*xhtml.Node {
	var tables []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.ElementNode && node.Data == "table" {
			tables = append(tables, node)
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(root)
	return tables
}

// wrapTable moves one rendered table into the host plugin-block fallback wrapper.
func wrapTable(table *xhtml.Node) {
	block := &xhtml.Node{
		Type: xhtml.ElementNode,
		Data: "div",
		Attr: []xhtml.Attribute{
			{Key: "class", Val: "kumbuka-plugin-block"},
			{Key: "data-kumbuka-plugin", Val: "me.kumbuka.tables"},
			{Key: "data-kumbuka-input", Val: "html"},
		},
	}
	classes := " " + htmlutil.Attribute(table, "class") + " "
	if strings.Contains(classes, " kumbuka-table-sortable ") || strings.Contains(classes, " kumbuka-table-filterable ") {
		block.Attr = append(block.Attr, xhtml.Attribute{Key: "data-kumbuka-module", Val: "interactive"})
	}

	parent := table.Parent
	parent.InsertBefore(block, table)
	parent.RemoveChild(table)
	fallback := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", Attr: []xhtml.Attribute{{Key: "data-kumbuka-fallback", Val: ""}}}
	block.AppendChild(fallback)
	fallback.AppendChild(table)
}
