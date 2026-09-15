package main

import (
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {}

func init() {
	sdk.RegisterModule("directives", transform)
	sdk.RegisterModule("presentation", transform)
}
func transform(request sdk.RenderRequest) sdk.RenderResult {
	enabled := func(name string) bool { v, ok := request.Features["me.kumbuka.tables."+name]; return !ok || v }
	options := tableOptions{Tables: enabled("tables"), TableStyles: enabled("styles"), TableSorting: enabled("sorting"), TableFiltering: enabled("filtering")}
	output := request.Source
	switch request.Stage {
	case "preprocess":
		if tableDirectivesEnabled(options) {
			output = preprocessTableDirectives(output, options)
		}
	case "postprocess":
		var err error
		if tableDirectivesEnabled(options) {
			output, err = applyTableDirectiveMarkers(output, options)
			if err != nil {
				return sdk.RenderResult{Error: err.Error()}
			}
		}
		output, err = markTables(output)
		if err != nil {
			return sdk.RenderResult{Error: err.Error()}
		}
	default:
		return sdk.RenderResult{Error: "unsupported stage"}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}

// markTables leaves semantic HTML as the accessible, script-free fallback.
func markTables(source string) (string, error) {
	nodes, err := xhtml.ParseFragment(strings.NewReader(source), &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return "", err
	}
	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	var tables []*xhtml.Node
	var walk func(*xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode && n.Data == "table" {
			tables = append(tables, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(root)
	for _, table := range tables {
		block := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []xhtml.Attribute{{Key: "class", Val: "kumbuka-plugin-block"}, {Key: "data-kumbuka-plugin", Val: "me.kumbuka.tables"}, {Key: "data-kumbuka-input", Val: "html"}}}
		classes := " " + htmlAttribute(table, "class") + " "
		if strings.Contains(classes, " kumbuka-table-sortable ") || strings.Contains(classes, " kumbuka-table-filterable ") {
			block.Attr = append(block.Attr, xhtml.Attribute{Key: "data-kumbuka-module", Val: "interactive"})
		}
		table.Parent.InsertBefore(block, table)
		table.Parent.RemoveChild(table)
		fallback := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div, Attr: []xhtml.Attribute{{Key: "data-kumbuka-fallback", Val: ""}}}
		block.AppendChild(fallback)
		fallback.AppendChild(table)
	}
	var out strings.Builder
	for n := root.FirstChild; n != nil; n = n.NextSibling {
		if err := xhtml.Render(&out, n); err != nil {
			return "", err
		}
	}
	return out.String(), nil
}
