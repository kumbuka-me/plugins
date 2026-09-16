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

	render := newRenderer(nodes, func(name string, size int) (template.HTML, error) {
		value, err := sdk.Icon(name, size)
		return template.HTML(value), err
	})
	html, err := render(options)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(html), nil
}
