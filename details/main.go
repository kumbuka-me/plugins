package main

import (
	sdk "github.com/kumbuka-me/sdk"
)

// main runs the package entry point.
func main() {}

// init registers the Details renderer with the Kumbuka plugin SDK.
func init() { sdk.RegisterModule("details", transform) }

// transform converts collapsible detail blocks before the core Markdown parser runs.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}

	return sdk.RenderResult{Parts: transformDetails(request.Source)}
}
