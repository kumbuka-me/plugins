package main

import (
	"context"

	sdk "github.com/kumbuka-me/sdk"
)

// main runs the package entry point.
func main() {}

// init registers the Page Report macro with the Kumbuka plugin SDK.
func init() { sdk.RegisterMacro("page-report", parse, renderMacro) }

// pages adapts the Kumbuka page capability to the page-report feature interface.
type pages struct {
}

// Search invokes Kumbuka's page-search host capability.
func (pages) Search(_ context.Context, query string, limit int) ([]sdk.Page, error) {
	return sdk.Pages().Search(sdk.PageQuery{Query: query, Limit: limit})
}

// GetPage invokes Kumbuka's page lookup host capability.
func (pages) GetPage(_ context.Context, slug string) (sdk.Page, error) {
	return sdk.Pages().Get(slug)
}

// renderMacro renders a parsed report using public page capabilities.
func renderMacro(options macroOptions) (sdk.Result, error) {
	html, err := newRenderer(context.Background(), pages{})(options)
	return sdk.Text(html), err
}
