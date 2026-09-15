package main

import (
	_ "embed"
	"html/template"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

// renderData contains one shared subpages template invocation.
type renderData struct {
	// Children contains prepared child navigation nodes.
	Children []sdk.NavigationNode
	// Title is the optional visible subpages heading.
	Title string
	// ShowTitle controls whether the heading is emitted.
	ShowTitle bool
}

//go:embed template.gohtml
var templateSource string

// newTemplate parses and returns the subpages rendering template.
func newTemplate(icon func(string, int) template.HTML) *template.Template {
	return template.Must(
		template.New("subpages").
			Funcs(template.FuncMap{
				"icon": icon,
			}).
			Parse(templateSource),
	)
}

// newRenderer returns a renderer for one prepared navigation subtree and URL strategy.
func newRenderer(nodes []sdk.NavigationNode, icon func(string, int) template.HTML) func(macroOptions) (string, error) {
	htmlTemplate := newTemplate(icon)
	return func(options macroOptions) (string, error) {
		if len(nodes) == 0 {
			return "", nil
		}

		var output strings.Builder
		err := htmlTemplate.ExecuteTemplate(&output, "subpage-toc", renderData{
			Children:  nodes,
			Title:     options.Title,
			ShowTitle: options.ShowTitle,
		})
		if err != nil {
			return "", err
		}
		return output.String(), nil
	}
}
