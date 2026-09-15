package main

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"strings"

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
			URL:   "/revisions/" + pagePath(context.Page.Slug),
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
	output.WriteString(`<div class="widget-entry"><div class="widget-entry-heading"><strong>` + html.EscapeString(revision.Author) + `</strong><span class="widget-badge">r` + fmt.Sprint(revision.Number) + `</span></div>`)
	if revision.Message != "" {
		output.WriteString(`<span class="widget-message">` + html.EscapeString(revision.Message) + `</span>`)
	}
	output.WriteString(`<span class="widget-meta"><span>` + revision.CreatedAt.Format("2006-01-02 15:04") + `</span>`)
	if revision.AddedLines != 0 || revision.RemovedLines != 0 {
		output.WriteString(`<span class="widget-diff"><span class="widget-added">+` + fmt.Sprint(revision.AddedLines) + `</span><span class="widget-removed">−` + fmt.Sprint(revision.RemovedLines) + `</span> lines</span>`)
	} else if revision.Number == 1 {
		output.WriteString(`<span>created page</span>`)
	} else {
		output.WriteString(`<span>metadata-only save</span>`)
	}
	output.WriteString("</span></div>")
	return output.String()
}

func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
