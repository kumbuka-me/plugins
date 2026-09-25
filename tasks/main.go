// Package main implements persistent actionable tasks for Kumbuka pages.
package main

import (
	"fmt"

	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers task rendering and page-details task controls.
func init() {
	sdk.RegisterModule("task", transform)
	sdk.RegisterWidgetWithCommands("page-details", renderWidget, commandWidget)
}

// transform replaces task declarations outside Markdown code with rendered task controls.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}
	return sdk.Text(transformSource(request.Source, sdk.Storage().Get))
}

// renderWidget renders task controls for declarations on the current page.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil || context.Page.Slug == "" {
		return sdk.Text(""), nil
	}
	content, err := sdk.Pages().Content(context.Page.Slug)
	if err != nil {
		return sdk.Result{}, err
	}
	return renderControls(content.Markdown, sdk.Storage().Get), nil
}

// commandWidget validates a page-scoped task action before persisting the new state.
func commandWidget(context sdk.WidgetCommandContext) (sdk.WidgetCommandResult, error) {
	if context.Page == nil || context.Page.Slug == "" {
		return sdk.WidgetCommandResult{}, fmt.Errorf("task command requires a page")
	}
	content, err := sdk.Pages().Content(context.Page.Slug)
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	taskID, done, ok := resolveAction(content.Markdown, context.Action)
	if !ok {
		return sdk.WidgetCommandResult{}, fmt.Errorf("task action is no longer available")
	}
	value := []byte("open")
	if done {
		value = []byte("done")
	}
	if err := sdk.Storage().Set(storageKey(taskID), value); err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	return sdk.WidgetCommandResult{Redirect: "/pages/" + pagePath(context.Page.Slug)}, nil
}
