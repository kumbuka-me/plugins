package main

import (
	"errors"
	"fmt"
	"html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

func main() {}

func init() { sdk.RegisterWidget("page-details", renderWidget) }

func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil {
		return sdk.Result{}, errors.New("revision history requires a page")
	}

	history, err := sdk.Pages().Revisions(sdk.RevisionQuery{Slug: context.Page.Slug, Limit: 1})
	if err != nil {
		return sdk.Result{}, err
	}
	result := sdk.Text(renderRevision(history))
	if history.Count > 1 {
		result.Actions = []sdk.WidgetAction{{
			ID:    "all",
			Kind:  "dialog",
			Label: fmt.Sprintf("All %d revisions", history.Count),
			URL:   "/revisions/" + widgetui.PagePath(context.Page.Slug),
			Icon:  "history-lucide",
		}}
	}
	return result, nil
}

func renderRevision(history sdk.RevisionHistory) string {
	var output strings.Builder
	output.WriteString("<h2>Revision history</h2>")
	if len(history.Revisions) == 0 {
		output.WriteString(`<p class="muted">No revision history available.</p>`)
		return output.String()
	}

	revision := history.Revisions[0]
	output.WriteString(`<div class="widget-entry"><div class="widget-entry-heading"><strong>`)
	output.WriteString(html.EscapeString(revision.Author))
	output.WriteString(`</strong><span class="widget-badge">r`)
	fmt.Fprint(&output, revision.Number)
	output.WriteString(`</span></div>`)

	if revision.Message != "" {
		output.WriteString(`<span class="widget-message">`)
		output.WriteString(html.EscapeString(revision.Message))
		output.WriteString(`</span>`)
	}

	output.WriteString(`<span class="widget-meta"><span>`)
	output.WriteString(revision.CreatedAt.Format("2006-01-02 15:04"))
	output.WriteString(`</span>`)

	if revision.AddedLines != 0 || revision.RemovedLines != 0 {
		output.WriteString(`<span class="widget-diff"><span class="widget-added">+`)
		fmt.Fprint(&output, revision.AddedLines)
		output.WriteString(`</span><span class="widget-removed">−`)
		fmt.Fprint(&output, revision.RemovedLines)
		output.WriteString(`</span> lines</span>`)
	} else if revision.Number == 1 {
		output.WriteString(`<span>created page</span>`)
	} else {
		output.WriteString(`<span>metadata-only save</span>`)
	}
	output.WriteString("</span></div>")
	return output.String()
}
