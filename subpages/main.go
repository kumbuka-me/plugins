// Package main implements the Subpages plugin.
package main

import (
	_ "embed"
	"html/template"
	"strconv"
	"strings"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
)

const defaultTitle = "Pages in this section"

type macroOptions struct {
	Title        string
	ShowTitle    bool
	DefaultTitle bool
}

type renderData struct {
	Children  []sdk.NavigationNode
	Title     string
	ShowTitle bool
	AriaLabel string
}

//go:embed template.gohtml
var templateSource string

func main() {}

func init() { sdk.RegisterLocalizedMacro("subpages", parse, renderMacro) }

func parse(line string) (macroOptions, bool) {
	value := strings.TrimSpace(line)
	if value == "{{subpages}}" {
		return macroOptions{Title: defaultTitle, ShowTitle: true, DefaultTitle: true}, true
	}

	argument, ok := strings.CutPrefix(value, "{{subpages ")
	if !ok {
		return macroOptions{}, false
	}
	argument, ok = strings.CutSuffix(argument, "}}")
	if !ok {
		return macroOptions{}, false
	}

	name, encodedTitle, ok := strings.Cut(strings.TrimSpace(argument), "=")
	if !ok || strings.TrimSpace(name) != "title" {
		return macroOptions{}, false
	}

	encodedTitle = strings.TrimSpace(encodedTitle)
	if len(encodedTitle) < 2 || encodedTitle[0] != '"' || encodedTitle[len(encodedTitle)-1] != '"' {
		return macroOptions{}, false
	}

	title, err := strconv.Unquote(encodedTitle)
	if err != nil {
		return macroOptions{}, false
	}
	return macroOptions{Title: title, ShowTitle: title != ""}, true
}

func renderMacro(invocation sdk.LocalizedMacro[macroOptions]) (sdk.Result, error) {
	options := invocation.Value
	localizer := localize.For(invocation.Locale)
	if options.DefaultTitle {
		options.Title = localizer.Text("subpages.title")
	}

	nodes, err := sdk.Pages().Navigation()
	if err != nil {
		return sdk.Result{}, err
	}

	html, err := renderSubpages(nodes, options, localizer, func(name string, size int) (template.HTML, error) {
		value, err := sdk.Icon(name, size)
		return template.HTML(value), err
	})
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(html), nil
}

func renderSubpages(
	nodes []sdk.NavigationNode,
	options macroOptions,
	localizer sdk.Localizer,
	icon func(string, int) (template.HTML, error),
) (string, error) {
	if len(nodes) == 0 {
		return "", nil
	}

	htmlTemplate := template.Must(template.New("subpages").Funcs(template.FuncMap{"icon": icon}).Parse(templateSource))
	var output strings.Builder
	if err := htmlTemplate.ExecuteTemplate(&output, "subpage-toc", renderData{
		Children:  nodes,
		Title:     options.Title,
		ShowTitle: options.ShowTitle,
		AriaLabel: localizer.Text("subpages.title"),
	}); err != nil {
		return "", err
	}
	return output.String(), nil
}
