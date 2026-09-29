package main

import (
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Tables postprocessor.
func init() { sdk.RegisterModule("presentation", transform) }

// transform applies request-scoped table directives and presentation wrapping after Markdown rendering.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "presentation" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported tables render request"}
	}

	output, err := processRenderedTables(request.Source, tableOptionsFromFeatures(request.Features), true)
	if err != nil {
		return sdk.Failure(err)
	}
	return sdk.Text(output)
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
