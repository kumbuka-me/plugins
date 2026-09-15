package main

import (
	"html/template"

	sdk "github.com/kumbuka-me/sdk"
)

func main() {}
func init() { sdk.RegisterMacro("subpages", parse, renderMacro) }
func renderMacro(options macroOptions) (sdk.Result, error) {
	nodes, err := sdk.Pages().Navigation()
	if err != nil {
		return sdk.Result{}, err
	}
	var iconError error
	render := newRenderer(nodes, func(name string, size int) template.HTML {
		value, err := sdk.Icon(name, size)
		if err != nil {
			iconError = err
		}
		return template.HTML(value)
	})
	html, err := render(options)
	if err != nil {
		return sdk.Result{}, err
	}
	if iconError != nil {
		return sdk.Result{}, iconError
	}
	return sdk.Text(html), nil
}
