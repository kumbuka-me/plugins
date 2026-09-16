package main

import (
	"html"
	"strings"
	"time"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

func main() {}

func init() { sdk.RegisterWidget("home", renderWidget) }

func renderWidget(sdk.WidgetContext) (sdk.Result, error) {
	drafts, err := sdk.Drafts().List(6)
	if err != nil {
		return sdk.Result{}, err
	}
	edits, err := sdk.Pages().RecentEdited(6)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderContinueWorking(drafts, edits, time.Now(), widgetui.HostIcon)), nil
}

func renderContinueWorking(drafts []sdk.PageDraft, edits []sdk.RecentEdit, now time.Time, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2 class="heading-with-icon">`)
	output.WriteString(icon("pencil-line-lucide", 15))
	output.WriteString(`<span>Continue working</span></h2></div><div class="widget-columns"><div><h3 class="widget-section-title">Drafts</h3>`)
	if len(drafts) == 0 {
		output.WriteString(`<p class="muted">No private drafts.</p>`)
	} else {
		for _, draft := range drafts {
			writeDraft(&output, draft, now, icon)
		}
	}
	output.WriteString(`</div><div><h3 class="widget-section-title">Recent edits</h3>`)
	if len(edits) == 0 {
		output.WriteString(`<p class="muted">Pages you edit will appear here.</p>`)
	} else {
		for _, edit := range edits {
			writeEdit(&output, edit, now, icon)
		}
	}
	output.WriteString(`</div></div>`)
	return output.String()
}

func writeDraft(output *strings.Builder, draft sdk.PageDraft, now time.Time, icon widgetui.IconRenderer) {
	title := draft.Title
	if title == "" {
		title = "Untitled"
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
	output.WriteString(`</strong><small>Private draft · `)
	output.WriteString(widgetui.RelativeTime(draft.UpdatedAt, now))
	if draft.Stale {
		output.WriteString(` · Page changed since draft started`)
	}
	output.WriteString(`</small></span></a>`)
}

func writeEdit(output *strings.Builder, edit sdk.RecentEdit, now time.Time, icon widgetui.IconRenderer) {
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
	output.WriteString(widgetui.RelativeTime(edit.UpdatedAt, now))
	output.WriteString(`</small></span></a>`)
}
