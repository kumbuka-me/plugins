package main

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	"github.com/kumbuka-me/plugins/internal/listindent"
	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// main provides the entry point required by the plugin executable.
func main() {}

// init registers the plugin's render and command handlers with the SDK.
func init() {
	sdk.RegisterModule("presentation", transform)
	sdk.RegisterWidgetWithCommands("commands", func(sdk.WidgetContext) (sdk.Result, error) { return sdk.Text(""), nil }, commandChecklist)
}

// transform converts rendered task lists into interactive checklist controls.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "presentation" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported checklist render request"}
	}
	output, err := presentChecklists(request.Source, listindent.Load(sdk.Settings().Get))
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}

// commandChecklist validates and applies a version-guarded checklist toggle.
func commandChecklist(context sdk.WidgetCommandContext) (sdk.WidgetCommandResult, error) {
	if context.Page == nil || context.Page.Slug == "" {
		return sdk.WidgetCommandResult{}, fmt.Errorf("checklist command requires a page")
	}
	matches := checklistAction.FindStringSubmatch(context.Action)
	if matches == nil {
		return sdk.WidgetCommandResult{}, fmt.Errorf("invalid checklist action")
	}
	index, err := strconv.Atoi(matches[1])
	if err != nil {
		return sdk.WidgetCommandResult{}, fmt.Errorf("invalid checklist item")
	}
	content, err := sdk.Pages().Content(context.Page.Slug)
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	markdown, err := toggleTaskMarker(content.Markdown, index, matches[2] == "1", matches[3])
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	_, err = sdk.Pages().UpdateContent(sdk.PageContentUpdate{
		Slug: context.Page.Slug, Markdown: markdown, Message: "Toggle checklist item",
		ExpectedUpdatedAt: content.UpdatedAt,
	})
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	return sdk.WidgetCommandResult{Redirect: "/pages/" + context.Page.Slug}, nil
}

// checklistSpan identifies one complete task-list fragment.
type checklistSpan struct {
	// start and end delimit the task-list fragment in the original HTML bytes.
	start, end int
	// firstIndex is the document-wide index of the first checkbox in the fragment.
	firstIndex int
}

// openElement tracks a pending HTML element during task-list scanning.
type openElement struct {
	// name is the open element's HTML tag name.
	name string
	// start is the opening tag's byte offset in the original HTML.
	start int
	// task reports whether this element contains a task checkbox.
	task bool
	// firstIndex is the first descendant checkbox's document-wide index.
	firstIndex int
}

// presentChecklists wraps each outer task list once so one sandboxed browser module can own every checkbox in the list without creating an iframe per row.
func presentChecklists(source, indent string) (string, error) {
	spans, fingerprint, err := checklistSpans(source)
	if err != nil || len(spans) == 0 {
		return source, err
	}
	var output strings.Builder
	output.Grow(len(source) + len(spans)*160)
	cursor := 0
	for _, span := range spans {
		output.WriteString(source[cursor:span.start])
		fragment, transformErr := presentChecklistFragment(source[span.start:span.end], span.firstIndex, fingerprint, listindent.Class("checklist", indent))
		if transformErr != nil {
			return "", transformErr
		}
		output.WriteString(`<div class="kumbuka-checklist-browser" data-kumbuka-plugin="me.kumbuka.task-lists" data-kumbuka-module="checklist-ui" data-kumbuka-input="html"><div class="kumbuka-checklist" data-kumbuka-fallback>`)
		output.WriteString(fragment)
		output.WriteString(`</div></div>`)
		cursor = span.end
	}
	output.WriteString(source[cursor:])
	return output.String(), nil
}

// checklistSpans locates outer task-list fragments and fingerprints their checkbox states.
func checklistSpans(source string) ([]checklistSpan, string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var stack []openElement
	var spans []checklistSpan
	var states []bool
	offset, checkboxIndex := 0, 0
	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return nil, "", tokenizer.Err()
		}
		raw := tokenizer.Raw()
		start := offset
		offset += len(raw)
		switch tokenType {
		case xhtml.StartTagToken, xhtml.SelfClosingTagToken:
			token := tokenizer.Token()
			if token.Data == "input" && isTaskCheckboxToken(token) && len(stack) != 0 && stack[len(stack)-1].name == "li" {
				stack[len(stack)-1].task = true
				for index := len(stack) - 1; index >= 0; index-- {
					if stack[index].name == "ul" || stack[index].name == "ol" {
						if !stack[index].task {
							stack[index].firstIndex = checkboxIndex
						}
						stack[index].task = true
					}
				}
				states = append(states, taskCheckboxCheckedToken(token))
				checkboxIndex++
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
				item := stack[index]
				if (item.name == "ul" || item.name == "ol") && item.task {
					spans = append(spans, checklistSpan{start: item.start, end: offset, firstIndex: item.firstIndex})
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
		if len(filtered) > 0 && span.start >= filtered[len(filtered)-1].start && span.end <= filtered[len(filtered)-1].end {
			continue
		}
		filtered = append(filtered, span)
	}
	return filtered, checklistFingerprint(states), nil
}

// presentChecklistFragment converts a task-list fragment into interactive checkbox controls.
func presentChecklistFragment(source string, firstIndex int, fingerprint, indentClass string) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}
	index := firstIndex
	walkChecklist(root, &index, fingerprint)
	markChecklistLists(root, indentClass)
	return htmlutil.RenderChildren(root)
}

// markChecklistLists applies the configured indentation class to nested list elements.
func markChecklistLists(node *xhtml.Node, className string) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == xhtml.ElementNode && (child.DataAtom == atom.Ul || child.DataAtom == atom.Ol) {
			htmlutil.AddClass(child, className)
		}
		markChecklistLists(child, className)
	}
}

// taskCheckboxCheckedToken reports whether an input token has a checked attribute.
func taskCheckboxCheckedToken(token xhtml.Token) bool {
	for _, attribute := range token.Attr {
		if attribute.Key == "checked" {
			return true
		}
	}
	return false
}

// isTaskCheckboxToken recognizes disabled checkbox inputs emitted by Markdown task lists.
func isTaskCheckboxToken(token xhtml.Token) bool {
	checkbox, disabled := false, false
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

// htmlVoidElement reports whether an HTML element has no closing tag.
func htmlVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

// walkChecklist replaces task checkboxes in document order without revisiting new controls.
func walkChecklist(node *xhtml.Node, index *int, fingerprint string) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if isTaskCheckbox(child) {
			replaceTaskCheckbox(child, *index, fingerprint)
			*index = *index + 1
		} else {
			walkChecklist(child, index, fingerprint)
		}
		child = next
	}
}

// isTaskCheckbox recognizes a disabled checkbox directly inside a list item.
func isTaskCheckbox(node *xhtml.Node) bool {
	return node.Type == xhtml.ElementNode && node.DataAtom == atom.Input && node.Parent != nil &&
		node.Parent.DataAtom == atom.Li && strings.EqualFold(htmlutil.Attribute(node, "type"), "checkbox") &&
		htmlutil.HasAttribute(node, "disabled")
}

// replaceTaskCheckbox replaces a task input with an accessible version-bound command button.
func replaceTaskCheckbox(node *xhtml.Node, index int, fingerprint string) {
	checked := htmlutil.HasAttribute(node, "checked")
	state, mark, label := "0", "✓", "Mark complete"
	className := "checklist-checkbox"
	if checked {
		state, label = "1", "Mark incomplete"
		className += " checked"
	}
	action := fmt.Sprintf("toggle-%d-%s-%s", index, state, fingerprint)
	className += " checklist-action__" + action
	button := &xhtml.Node{Type: xhtml.ElementNode, Data: "button", DataAtom: atom.Button, Attr: []xhtml.Attribute{
		{Key: "type", Val: "button"}, {Key: "class", Val: className}, {Key: "role", Val: "checkbox"},
		{Key: "aria-checked", Val: strconv.FormatBool(checked)}, {Key: "aria-label", Val: label},
	}}
	button.AppendChild(&xhtml.Node{Type: xhtml.TextNode, Data: mark})
	htmlutil.AddClass(node.Parent, "checklist-item")
	node.Parent.InsertBefore(button, node)
	node.Parent.RemoveChild(node)
}
