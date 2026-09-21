// Package main implements persistent, configurable inline status badges for Kumbuka.
package main

import (
	"fmt"

	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the status renderer and page-details controls.
func init() {
	sdk.RegisterModule("status", transform)
	sdk.RegisterWidgetWithCommands("page-details", renderWidget, commandWidget)
}

// transform replaces status declarations outside Markdown code with rendered badges.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}

	output := transformSource(request.Source, sdk.Resources().Get, sdk.Storage().Get)
	return sdk.Text(output)
}

// renderWidget renders status controls for declarations on the current page.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil || context.Page.Slug == "" {
		return sdk.Text(""), nil
	}

	content, err := sdk.Pages().Content(context.Page.Slug)
	if err != nil {
		return sdk.Result{}, err
	}

	return renderControls(content.Markdown, sdk.Resources().Get, sdk.Storage().Get), nil
}

// commandWidget validates a page-scoped status action before persisting the selected value.
func commandWidget(context sdk.WidgetCommandContext) (sdk.WidgetCommandResult, error) {
	if context.Page == nil || context.Page.Slug == "" {
		return sdk.WidgetCommandResult{}, fmt.Errorf("status command requires a page")
	}

	content, err := sdk.Pages().Content(context.Page.Slug)
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}

	statusID, value, ok := resolveAction(content.Markdown, context.Action, sdk.Resources().Get)
	if !ok {
		return sdk.WidgetCommandResult{}, fmt.Errorf("status action is no longer available")
	}
	if err := sdk.Storage().Set(storageKey(statusID), []byte(value)); err != nil {
		return sdk.WidgetCommandResult{}, err
	}

	return sdk.WidgetCommandResult{Redirect: "/pages/" + pagePath(context.Page.Slug)}, nil
}
