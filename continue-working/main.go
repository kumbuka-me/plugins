package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

type iconRenderer func(string, int) string

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
	return sdk.Text(renderContinueWorking(drafts, edits, time.Now(), hostIcon)), nil
}

func renderContinueWorking(drafts []sdk.PageDraft, edits []sdk.RecentEdit, now time.Time, icon iconRenderer) string {
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

func writeDraft(output *strings.Builder, draft sdk.PageDraft, now time.Time, icon iconRenderer) {
	title := draft.Title
	if title == "" {
		title = "Untitled"
	}
	editURL := "/pages/new"
	if draft.PageID > 0 && draft.PageSlug != "" {
		editURL = "/edit/" + pagePath(draft.PageSlug)
	}
	output.WriteString(`<a class="widget-item" href="` + editURL + `"><span class="widget-item-icon">`)
	output.WriteString(icon("pencil-line-lucide", 16))
	output.WriteString(`</span><span><strong>` + html.EscapeString(title) + `</strong><small>Private draft · ` + relativeTime(draft.UpdatedAt, now))
	if draft.Stale {
		output.WriteString(` · Page changed since draft started`)
	}
	output.WriteString(`</small></span></a>`)
}

func writeEdit(output *strings.Builder, edit sdk.RecentEdit, now time.Time, icon iconRenderer) {
	output.WriteString(`<a class="widget-item" href="/edit/` + pagePath(edit.Slug) + `"><span class="widget-item-icon">`)
	output.WriteString(pageIcon(edit.Page, 16, icon))
	output.WriteString(`</span><span><strong>` + html.EscapeString(edit.Title) + `</strong><small>`)
	if edit.RevisionMessage != "" {
		output.WriteString(html.EscapeString(edit.RevisionMessage) + ` · `)
	}
	output.WriteString(relativeTime(edit.UpdatedAt, now) + `</small></span></a>`)
}

func pageIcon(page sdk.Page, size int, icon iconRenderer) string {
	if page.Icon != "" {
		if rendered := icon(page.Icon, size); rendered != "" {
			return rendered
		}
	}
	return icon("file-text-lucide", size)
}

func relativeTime(value, now time.Time) string {
	if value.IsZero() {
		return ""
	}
	delta := now.Sub(value)
	if delta < 0 || delta < time.Minute {
		return "just now"
	}
	if delta < time.Hour {
		return fmt.Sprintf("%dm ago", int(delta/time.Minute))
	}
	if delta < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(delta/time.Hour))
	}
	return fmt.Sprintf("%dd ago", int(delta/(24*time.Hour)))
}

func hostIcon(name string, size int) string {
	rendered, err := sdk.Icon(name, size)
	if err != nil {
		return ""
	}
	return rendered
}

func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
