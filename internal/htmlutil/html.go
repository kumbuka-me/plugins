// Package htmlutil provides shared DOM helpers for first-party HTML postprocessors.
package htmlutil

import (
	"strings"

	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ParseFragment parses an HTML fragment below a synthetic div root.
func ParseFragment(source string) (*xhtml.Node, error) {
	contextNode := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	nodes, err := xhtml.ParseFragment(strings.NewReader(source), contextNode)
	if err != nil {
		return nil, err
	}

	root := &xhtml.Node{Type: xhtml.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, node := range nodes {
		root.AppendChild(node)
	}
	return root, nil
}

// RenderChildren serializes every direct child of a synthetic fragment root.
func RenderChildren(root *xhtml.Node) (string, error) {
	var output strings.Builder
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if err := xhtml.Render(&output, node); err != nil {
			return "", err
		}
	}
	return output.String(), nil
}

// Attribute returns one HTML node attribute by key.
func Attribute(node *xhtml.Node, key string) string {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

// HasAttribute reports whether an HTML node contains an attribute with the given key.
func HasAttribute(node *xhtml.Node, key string) bool {
	for _, attribute := range node.Attr {
		if attribute.Key == key {
			return true
		}
	}
	return false
}

// SetAttribute replaces or appends one HTML node attribute.
func SetAttribute(node *xhtml.Node, key, value string) {
	for index := range node.Attr {
		if node.Attr[index].Key == key {
			node.Attr[index].Val = value
			return
		}
	}
	node.Attr = append(node.Attr, xhtml.Attribute{Key: key, Val: value})
}

// AddClass adds a class token when the element does not already contain it.
func AddClass(node *xhtml.Node, className string) {
	classes := strings.Fields(Attribute(node, "class"))
	for _, existing := range classes {
		if existing == className {
			return
		}
	}
	classes = append(classes, className)
	SetAttribute(node, "class", strings.Join(classes, " "))
}
