package main

import (
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func main() {}

func init() { sdk.RegisterModule("presentation", transform) }

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

// presentTaskLists replaces Goldmark's disabled checkbox controls with inert,
// accessible text markers owned entirely by this plugin.
func presentTaskLists(source string) (string, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(source), contextNode)
	if err != nil {
		return "", err
	}

	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, node := range nodes {
		root.AppendChild(node)
	}

	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		for child := node.FirstChild; child != nil; {
			next := child.NextSibling
			if isTaskCheckbox(child) {
				replaceTaskCheckbox(child)
			} else {
				walk(child)
			}
			child = next
		}
	}
	walk(root)

	var output strings.Builder
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if err := xhtml.Render(&output, node); err != nil {
			return "", err
		}
	}
	return output.String(), nil
}

func isTaskCheckbox(node *xhtml.Node) bool {
	if node.Type != xhtml.ElementNode || node.DataAtom != atom.Input || node.Parent == nil || node.Parent.DataAtom != atom.Li {
		return false
	}
	return strings.EqualFold(attribute(node, "type"), "checkbox") && hasAttribute(node, "disabled")
}

func replaceTaskCheckbox(node *xhtml.Node) {
	checked := hasAttribute(node, "checked")
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
	addClass(parent, "task-list-item")
	parent.InsertBefore(span, node)
	parent.RemoveChild(node)
}

func attribute(node *xhtml.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func hasAttribute(node *xhtml.Node, key string) bool {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return true
		}
	}
	return false
}

func addClass(node *xhtml.Node, name string) {
	for index := range node.Attr {
		if node.Attr[index].Key != "class" {
			continue
		}
		for _, existing := range strings.Fields(node.Attr[index].Val) {
			if existing == name {
				return
			}
		}
		node.Attr[index].Val = strings.TrimSpace(node.Attr[index].Val + " " + name)
		return
	}
	node.Attr = append(node.Attr, xhtml.Attribute{Key: "class", Val: name})
}
