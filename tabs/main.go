package main

import (
	sdk "github.com/kumbuka-me/sdk"
)

// main runs the package entry point.
func main() {}

func init() { sdk.RegisterModule("tabs", transform) }

// transform converts Material-style tab groups before the core Markdown parser runs.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}

	return sdk.RenderResult{Parts: transformTabs(request.Source)}
}
