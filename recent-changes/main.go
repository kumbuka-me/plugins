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
	pages, err := sdk.Pages().Recent(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderRecent(pages, time.Now(), hostIcon)), nil
}

func renderRecent(pages []sdk.Page, now time.Time, icon iconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>Recent changes</h2><a href="/search">View all</a></div>`)
	if len(pages) == 0 {
		output.WriteString(`<div class="empty"><strong>Your Kumbuka knowledge base is ready.</strong><p>Create the first page and start linking knowledge together.</p></div>`)
		return output.String()
	}
	for _, page := range pages {
		writePageRow(&output, page, now, icon)
	}
	return output.String()
}

func writePageRow(output *strings.Builder, page sdk.Page, now time.Time, icon iconRenderer) {
	output.WriteString(`<a class="page-row" href="/pages/`)
	output.WriteString(pagePath(page.Slug))
	output.WriteString(`"><span class="doc-icon">`)
	output.WriteString(pageIcon(page, 17, icon))
	output.WriteString(`</span><span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	output.WriteString(html.EscapeString(page.Slug))
	if len(page.Tags) != 0 {
		output.WriteString(` · `)
		output.WriteString(html.EscapeString(strings.Join(page.Tags, ", ")))
	}
	output.WriteString(`</small></span><time>`)
	output.WriteString(relativeTime(page.UpdatedAt, now))
	output.WriteString(`</time></a>`)
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
