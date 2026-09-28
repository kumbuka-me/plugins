package main

import (
	"io"
	"sort"
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

// presentTaskLists replaces Goldmark's disabled checkbox controls with inert
// accessible text markers. It tokenizes the complete page but parses only list
// item fragments that actually contain task checkboxes.
func presentTaskLists(source string) (string, error) {
	spans, err := taskListItemSpans(source)
	if err != nil || len(spans) == 0 {
		return source, err
	}

	var output strings.Builder
	output.Grow(len(source))
	cursor := 0
	for _, span := range spans {
		output.WriteString(source[cursor:span.start])
		fragment, err := presentTaskListFragment(source[span.start:span.end])
		if err != nil {
			return "", err
		}
		output.WriteString(fragment)
		cursor = span.end
	}
	output.WriteString(source[cursor:])
	return output.String(), nil
}

// taskListSpan identifies one list-item byte range that needs transformation.
type taskListSpan struct {
	// start is the first byte in the list item.
	start int
	// end is the first byte after the list item.
	end int
}

// openElement tracks the minimal tokenizer state needed to identify direct-child checkboxes.
type openElement struct {
	// name is the lowercase HTML element name.
	name string
	// start is the opening tag byte offset.
	start int
	// task reports whether this list item owns a task checkbox.
	task bool
}

// taskListItemSpans locates list items whose direct children contain Goldmark
// task checkboxes. Nested task items are collapsed into their outer transformed
// fragment so ranges never overlap.
func taskListItemSpans(source string) ([]taskListSpan, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var stack []openElement
	var spans []taskListSpan
	offset := 0

	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return nil, tokenizer.Err()
		}

		raw := tokenizer.Raw()
		start := offset
		offset += len(raw)

		switch tokenType {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "input" && isTaskCheckboxToken(token) && len(stack) != 0 && stack[len(stack)-1].name == "li" {
				stack[len(stack)-1].task = true
			}
			if tokenType == xhtml.StartTagToken && !htmlVoidElement(token.Data) {
				stack = append(stack, openElement{name: token.Data, start: start})
			}

		case xhtml.EndTagToken:
			token := tokenizer.Token()
			for index := len(stack) - 1; index >= 0; index-- {
				if stack[index].name != token.Data {
					continue
				}
				if token.Data == "li" && stack[index].task {
					spans = append(spans, taskListSpan{start: stack[index].start, end: offset})
				}
				stack = stack[:index]
				break
			}
		}
	}

	sort.Slice(spans, func(i, j int) bool {
		if spans[i].start == spans[j].start {
			return spans[i].end > spans[j].end
		}
		return spans[i].start < spans[j].start
	})
	filtered := spans[:0]
	for _, span := range spans {
		if len(filtered) != 0 {
			previous := filtered[len(filtered)-1]
			if span.start >= previous.start && span.end <= previous.end {
				continue
			}
		}
		filtered = append(filtered, span)
	}
	return filtered, nil
}

// presentTaskListFragment transforms one bounded list-item fragment.
func presentTaskListFragment(source string) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}
	walkTaskList(root)
	return htmlutil.RenderChildren(root)
}

// isTaskCheckboxToken reports whether a tokenizer token is a disabled checkbox.
func isTaskCheckboxToken(token xhtml.Token) bool {
	checkbox := false
	disabled := false
	for _, attribute := range token.Attr {
		switch attribute.Key {
		case "type":
			checkbox = strings.EqualFold(attribute.Val, "checkbox")
		case "disabled":
			disabled = true
		}
	}
	return checkbox && disabled
}

// htmlVoidElement reports whether an HTML start tag never pushes an open element.
func htmlVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
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
