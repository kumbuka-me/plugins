// Package main implements persistent actionable tasks for Kumbuka pages.
package main

import (
	"fmt"

	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers task rendering, page-details controls, and committed assignment notifications.
func init() {
	sdk.RegisterModule("task", transform)
	sdk.RegisterWidgetWithCommands("page-details", renderWidget, commandWidget)
	sdk.RegisterContentChange("assignments", contentChanged)
}

// transform replaces task declarations outside Markdown code with rendered task controls.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Stage != "preprocess" {
		return sdk.RenderResult{Error: "unsupported stage"}
	}
	workflow, err := loadTaskWorkflow(sdk.Settings().Get)
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.Text(transformSource(request.Source, sdk.Storage().Get, workflow))
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
	workflow, err := loadTaskWorkflow(sdk.Settings().Get)
	if err != nil {
		return sdk.Result{}, err
	}
	return renderControls(content.Markdown, sdk.Storage().Get, workflow), nil
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
	workflow, err := loadTaskWorkflow(sdk.Settings().Get)
	if err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	if err := applyTaskAction(*context.Page, content.Markdown, context.Action, workflow, taskMutationServices{
		Read:             sdk.Storage().Get,
		Write:            sdk.Storage().Set,
		ResolveMention:   sdk.Users().ResolveMention,
		SendNotification: sdk.Notifications().Send,
	}); err != nil {
		return sdk.WidgetCommandResult{}, err
	}
	return sdk.WidgetCommandResult{Redirect: "/pages/" + pagePath(context.Page.Slug)}, nil
}

// contentChanged notifies users whose task assignment was added or changed by a committed page edit.
func contentChanged(context sdk.ContentChangeContext) error {
	return notifyTaskAssignments(context, taskAssignmentServices{
		ResolveMention:   sdk.Users().ResolveMention,
		SendNotification: sdk.Notifications().Send,
	})
}
