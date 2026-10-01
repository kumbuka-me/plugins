package main

import (
	"html"
	"strings"
	"time"

	"github.com/kumbuka-me/plugins/internal/localize"
	"github.com/kumbuka-me/plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Continue Working home widget.
func init() { sdk.RegisterWidget("home", renderWidget) }

// renderWidget loads the current viewer's drafts and recent edits for the home widget.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	drafts, err := sdk.Drafts().List(6)
	if err != nil {
		return sdk.Result{}, err
	}
	edits, err := sdk.Pages().RecentEdited(6)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderContinueWorking(drafts, edits, time.Now(), widgetui.HostIcon, localize.For(context.Locale))), nil
}

// renderContinueWorking renders the two-column drafts and recent-edits widget body.
func renderContinueWorking(drafts []sdk.PageDraft, edits []sdk.RecentEdit, now time.Time, icon widgetui.IconRenderer, localizer sdk.Localizer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2 class="heading-with-icon">`)
	output.WriteString(icon("pencil-line-lucide", 15))
	output.WriteString(`<span>`)
	output.WriteString(localizer.Text("continue.title"))
	output.WriteString(`</span></h2></div><div class="widget-columns"><div><h3 class="widget-section-title">`)
	output.WriteString(localizer.Text("continue.drafts"))
	output.WriteString(`</h3>`)
	if len(drafts) == 0 {
		output.WriteString(`<p class="muted">` + html.EscapeString(localizer.Text("continue.no_drafts")) + `</p>`)
	} else {
		for _, draft := range drafts {
			writeDraft(&output, draft, now, icon, localizer)
		}
	}
	output.WriteString(`</div><div><h3 class="widget-section-title">`)
	output.WriteString(localizer.Text("continue.edits"))
	output.WriteString(`</h3>`)
	if len(edits) == 0 {
		output.WriteString(`<p class="muted">` + html.EscapeString(localizer.Text("continue.no_edits")) + `</p>`)
	} else {
		for _, edit := range edits {
			writeEdit(&output, edit, now, icon, localizer)
		}
	}
	output.WriteString(`</div></div>`)
	return output.String()
}

// writeDraft appends one private draft entry to the widget markup.
func writeDraft(output *strings.Builder, draft sdk.PageDraft, now time.Time, icon widgetui.IconRenderer, localizer sdk.Localizer) {
	title := draft.Title
	if title == "" {
		title = localizer.Text("common.untitled")
	}
	editURL := "/pages/new"
	if draft.PageID > 0 && draft.PageSlug != "" {
		editURL = "/edit/" + widgetui.PagePath(draft.PageSlug)
	}
	output.WriteString(`<a class="widget-item" href="`)
	output.WriteString(editURL)
	output.WriteString(`"><span class="widget-item-icon">`)
	output.WriteString(icon("pencil-line-lucide", 16))
	output.WriteString(`</span><span><strong>`)
	output.WriteString(html.EscapeString(title))
	output.WriteString(`</strong><small>`)
	output.WriteString(html.EscapeString(localizer.Text("continue.private_draft")))
	output.WriteString(` · `)
	output.WriteString(widgetui.RelativeTime(draft.UpdatedAt, now, localizer))
	if draft.Stale {
		output.WriteString(` · ` + html.EscapeString(localizer.Text("continue.stale")))
	}
	output.WriteString(`</small></span></a>`)
}

// writeEdit appends one recently edited page entry to the widget markup.
func writeEdit(output *strings.Builder, edit sdk.RecentEdit, now time.Time, icon widgetui.IconRenderer, localizer sdk.Localizer) {
	output.WriteString(`<a class="widget-item" href="/edit/`)
	output.WriteString(widgetui.PagePath(edit.Slug))
	output.WriteString(`"><span class="widget-item-icon">`)
	output.WriteString(widgetui.PageIcon(edit.Page, 16, icon))
	output.WriteString(`</span><span><strong>`)
	output.WriteString(html.EscapeString(edit.Title))
	output.WriteString(`</strong><small>`)
	if edit.RevisionMessage != "" {
		output.WriteString(html.EscapeString(edit.RevisionMessage))
		output.WriteString(` · `)
	}
	output.WriteString(widgetui.RelativeTime(edit.UpdatedAt, now, localizer))
	output.WriteString(`</small></span></a>`)
}
