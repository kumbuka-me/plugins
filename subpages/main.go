package main

import (
	"html/template"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Subpages macro with the Kumbuka plugin SDK.
func init() { sdk.RegisterLocalizedMacro("subpages", parse, renderMacro) }

// renderMacro loads navigation data and renders one parsed Subpages invocation.
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

	render := newRenderer(nodes, localizer, func(name string, size int) (template.HTML, error) {
		value, err := sdk.Icon(name, size)
		return template.HTML(value), err
	})
	html, err := render(options)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(html), nil
}
