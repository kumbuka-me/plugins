package main

import (
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Task Lists presentation postprocessor.
func init() { sdk.RegisterModule("presentation", transform) }

// transform converts rendered task-list checkboxes into the plugin's inert presentation markup.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "presentation" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported task-list render request"}
	}

	output, err := presentTaskLists(request.Source)
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}

// presentTaskLists replaces Goldmark's disabled checkbox controls with inert accessible text markers.
func presentTaskLists(source string) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}

	walkTaskList(root)
	return htmlutil.RenderChildren(root)
}

// walkTaskList replaces task-list checkboxes below one DOM node.
func walkTaskList(node *xhtml.Node) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if isTaskCheckbox(child) {
			replaceTaskCheckbox(child)
		} else {
			walkTaskList(child)
		}
		child = next
	}
}

// isTaskCheckbox reports whether a node is a disabled checkbox directly inside a list item.
func isTaskCheckbox(node *xhtml.Node) bool {
	if node.Type != xhtml.ElementNode || node.DataAtom != atom.Input || node.Parent == nil || node.Parent.DataAtom != atom.Li {
		return false
	}
	return strings.EqualFold(htmlutil.Attribute(node, "type"), "checkbox") && htmlutil.HasAttribute(node, "disabled")
}

// replaceTaskCheckbox replaces one disabled input with an accessible inert checkbox marker.
func replaceTaskCheckbox(node *xhtml.Node) {
	checked := htmlutil.HasAttribute(node, "checked")
	className := "task-list-checkbox"
	mark := "☐"
	checkedValue := "false"
	if checked {
		className += " checked"
		mark = "☑"
		checkedValue = "true"
	}

	span := &xhtml.Node{
		Type:     xhtml.ElementNode,
		Data:     "span",
		DataAtom: atom.Span,
		Attr: []xhtml.Attribute{
			{Key: "class", Val: className},
			{Key: "role", Val: "checkbox"},
			{Key: "aria-checked", Val: checkedValue},
			{Key: "aria-disabled", Val: "true"},
		},
	}
	span.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: mark})

	parent := node.Parent
	htmlutil.AddClass(parent, "task-list-item")
	parent.InsertBefore(span, node)
	parent.RemoveChild(node)
}
