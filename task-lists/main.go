package main

import (
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/kumbuka-me/plugins/internal/htmlutil"
	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

var (
	checklistAction = regexp.MustCompile(`^toggle-([0-9]+)-([01])$`)
	taskMarker      = regexp.MustCompile(`^((?:[ \t]{0,3}>[ \t]?)*[ \t]*(?:[-+*]|[0-9]+[.)])[ \t]+\[)([ xX])(\])`)
)

func main() {}

func init() {
	sdk.RegisterModule("presentation", transform)
	sdk.RegisterWidgetWithCommands("commands", renderCommands, commandChecklist)
}

func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "presentation" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported checklist render request"}
	}
	output, err := presentChecklists(request.Source)
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.RenderResult{Parts: []sdk.RenderPart{{Text: output}}}
}

func renderCommands(sdk.WidgetContext) (sdk.Result, error) { return sdk.Text(""), nil }

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
	markdown, err := toggleTaskMarker(content.Markdown, index, matches[2] == "1")
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

type checklistSpan struct {
	start, end int
	firstIndex int
}

type openElement struct {
	name       string
	start      int
	task       bool
	firstIndex int
}

// presentChecklists wraps each outer task list once so one sandboxed browser
// module can own every checkbox in the list without creating an iframe per row.
func presentChecklists(source string) (string, error) {
	spans, err := checklistSpans(source)
	if err != nil || len(spans) == 0 {
		return source, err
	}
	var output strings.Builder
	output.Grow(len(source) + len(spans)*160)
	cursor := 0
	for _, span := range spans {
		output.WriteString(source[cursor:span.start])
		fragment, transformErr := presentChecklistFragment(source[span.start:span.end], span.firstIndex)
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

func checklistSpans(source string) ([]checklistSpan, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var stack []openElement
	var spans []checklistSpan
	offset, checkboxIndex := 0, 0
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
				for index := len(stack) - 1; index >= 0; index-- {
					if stack[index].name == "ul" || stack[index].name == "ol" {
						if !stack[index].task {
							stack[index].firstIndex = checkboxIndex
						}
						stack[index].task = true
					}
				}
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
	return filtered, nil
}

func presentChecklistFragment(source string, firstIndex int) (string, error) {
	root, err := htmlutil.ParseFragment(source)
	if err != nil {
		return "", err
	}
	index := firstIndex
	walkChecklist(root, &index)
	return htmlutil.RenderChildren(root)
}

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

func htmlVoidElement(name string) bool {
	switch name {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	default:
		return false
	}
}

func walkChecklist(node *xhtml.Node, index *int) {
	for child := node.FirstChild; child != nil; {
		next := child.NextSibling
		if isTaskCheckbox(child) {
			replaceTaskCheckbox(child, *index)
			*index = *index + 1
		} else {
			walkChecklist(child, index)
		}
		child = next
	}
}

func isTaskCheckbox(node *xhtml.Node) bool {
	return node.Type == xhtml.ElementNode && node.DataAtom == atom.Input && node.Parent != nil &&
		node.Parent.DataAtom == atom.Li && strings.EqualFold(htmlutil.Attribute(node, "type"), "checkbox") &&
		htmlutil.HasAttribute(node, "disabled")
}

func replaceTaskCheckbox(node *xhtml.Node, index int) {
	checked := htmlutil.HasAttribute(node, "checked")
	state, mark, label := "0", "✓", "Mark complete"
	className := "checklist-checkbox"
	if checked {
		state, label = "1", "Mark incomplete"
		className += " checked"
	}
	action := fmt.Sprintf("toggle-%d-%s", index, state)
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

// toggleTaskMarker changes one task marker in document order while ignoring
// fenced code blocks. expectedChecked rejects stale rendered controls.
func toggleTaskMarker(source string, target int, expectedChecked bool) (string, error) {
	lines := strings.SplitAfter(source, "\n")
	itemIndex := 0
	fence := byte(0)
	fenceLength := 0
	for lineIndex, line := range lines {
		content := strings.TrimSuffix(line, "\n")
		trimmed := strings.TrimLeft(content, " \t")
		if marker, length := fenceMarker(trimmed); marker != 0 {
			if fence == 0 {
				fence, fenceLength = marker, length
			} else if marker == fence && length >= fenceLength {
				fence, fenceLength = 0, 0
			}
			continue
		}
		if fence != 0 {
			continue
		}
		match := taskMarker.FindStringSubmatchIndex(content)
		if match == nil {
			continue
		}
		if itemIndex == target {
			marker := content[match[4]:match[5]]
			checked := marker == "x" || marker == "X"
			if checked != expectedChecked {
				return "", fmt.Errorf("checklist item changed; reload the page")
			}
			replacement := "x"
			if checked {
				replacement = " "
			}
			lines[lineIndex] = content[:match[4]] + replacement + content[match[5]:] + strings.TrimPrefix(line, content)
			return strings.Join(lines, ""), nil
		}
		itemIndex++
	}
	return "", fmt.Errorf("checklist item no longer exists")
}

func fenceMarker(line string) (byte, int) {
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, 0
	}
	marker := line[0]
	length := 1
	for length < len(line) && line[length] == marker {
		length++
	}
	if length < 3 {
		return 0, 0
	}
	return marker, length
}
