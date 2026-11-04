package main

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/kumbuka-me/plugins/internal/localize"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Revision History page-details widget.
func init() { sdk.RegisterWidget("page-details", renderWidget) }

// renderWidget loads the latest revision and exposes a host-owned comparison dialog action.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil {
		return sdk.Result{}, errors.New("revision history requires a page")
	}

	history, err := sdk.Pages().Revisions(sdk.RevisionQuery{Slug: context.Page.Slug, Limit: 1})
	if err != nil {
		return sdk.Result{}, err
	}
	return revisionResult(history, context.Page.Slug, localize.For(context.Locale)), nil
}

// revisionResult returns the compact widget and its host-rendered comparison action.
func revisionResult(history sdk.RevisionHistory, slug string, localizer sdk.Localizer) sdk.Result {
	result := sdk.Text(renderRevision(history, localizer))
	if len(history.Revisions) == 0 {
		return result
	}

	result.Actions = []sdk.WidgetAction{{
		ID:    "compare",
		Kind:  "dialog",
		Label: localizer.Text("revision.view_changes"),
		URL:   fmt.Sprintf("/revisions/%s?revision=%d", slug, history.Revisions[0].Number),
	}}
	return result
}

// renderRevision renders the latest version without repeating what the comparison dialog shows.
func renderRevision(history sdk.RevisionHistory, localizer sdk.Localizer) string {
	var output strings.Builder
	output.WriteString("<h2>" + localizer.Text("revision.title") + "</h2>")
	if len(history.Revisions) == 0 {
		output.WriteString(`<p class="muted">` + localizer.Text("revision.empty") + `</p>`)
		return output.String()
	}

	revision := history.Revisions[0]
	output.WriteString(`<div class="widget-entry widget-entry--clickable">`)
	output.WriteString(`<div class="widget-entry-heading"><strong>`)
	output.WriteString(html.EscapeString(revision.Author))
	output.WriteString(`</strong><span class="widget-revision-number">r`)
	fmt.Fprint(&output, revision.Number)
	output.WriteString(`</span></div>`)

	output.WriteString(`<div class="widget-revision-meta">`)
	if !revision.CreatedAt.IsZero() {
		output.WriteString(`<span class="widget-revision-date">`)
		output.WriteString(revisionDate(revision, localizer))
		output.WriteString(`</span>`)
	}
	if revision.AddedLines != 0 || revision.RemovedLines != 0 {
		output.WriteString(`<span class="widget-revision-diff"><span class="widget-added">+`)
		fmt.Fprint(&output, revision.AddedLines)
		output.WriteString(`</span><span class="widget-removed">−`)
		fmt.Fprint(&output, revision.RemovedLines)
		output.WriteString(`</span><span class="widget-revision-unit">`)
		output.WriteString(localizer.Text("revision.lines"))
		output.WriteString(`</span></span>`)
	} else {
		output.WriteString(`<span class="widget-revision-fields">`)
		output.WriteString(html.EscapeString(revisionFields(revision, localizer)))
		output.WriteString(`</span>`)
	}
	output.WriteString(`</div></div>`)
	return output.String()
}

// revisionDate formats one revision timestamp for the selected interface language.
func revisionDate(revision sdk.Revision, localizer sdk.Localizer) string {
	if localizer.Locale() == "de" {
		return revision.CreatedAt.Format("02.01.2006 · 15:04")
	}
	return revision.CreatedAt.Format("2006-01-02 · 15:04")
}

// revisionFields describes changed metadata when the Markdown comparison has no line changes.
func revisionFields(revision sdk.Revision, localizer sdk.Localizer) string {
	if revision.Number == 1 {
		return localizer.Text("revision.created")
	}

	// Core-generated messages contain field names after "Changed:". Custom messages
	// are not interpreted as field names, so they fall back to a neutral description.
	if _, fields, found := strings.Cut(revision.Message, "Changed: "); found {
		fields, _, _ = strings.Cut(fields, " · ")
		if fields = strings.TrimSpace(fields); fields != "" {
			return localizer.Text("revision.changed_fields") + ": " + fields
		}
	}
	return localizer.Text("revision.metadata")
}
