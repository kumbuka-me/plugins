package main

import (
	"errors"
	"html"
	"net/url"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

func main() {}

func init() { sdk.RegisterWidget("page-details", renderWidget) }

func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil {
		return sdk.Result{}, errors.New("wiki links requires a page")
	}

	links, err := sdk.Pages().Links(context.Page.Slug)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderLinks(links)), nil
}

func renderLinks(links sdk.PageLinks) string {
	var output strings.Builder
	output.WriteString("<h2>Referenced by</h2>")
	output.WriteString(`<div class="widget-list">`)
	if len(links.Backlinks) == 0 {
		output.WriteString(`<p class="muted">No pages link here yet.</p>`)
	} else {
		for _, page := range links.Backlinks {
			writePageRow(&output, page)
		}
	}
	output.WriteString("</div>")

	output.WriteString("<h2>Links from this page</h2>")
	output.WriteString(`<div class="widget-list">`)
	if len(links.Outgoing) == 0 {
		output.WriteString(`<p class="muted">No wiki links on this page.</p>`)
	} else {
		for _, link := range links.Outgoing {
			if link.Exists {
				title := link.TargetTitle
				if title == "" {
					title = link.TargetSlug
				}
				output.WriteString(`<a class="widget-row" href="/pages/` + pagePath(link.TargetSlug) + `"><strong>` + html.EscapeString(title) + `</strong><span class="widget-meta">` + html.EscapeString(link.TargetSlug) + `</span></a>`)
				continue
			}
			output.WriteString(`<a class="widget-row broken" href="/pages/new?slug=` + url.QueryEscape(link.TargetSlug) + `"><strong>` + html.EscapeString(link.TargetSlug) + `</strong><span class="widget-meta">Missing page</span></a>`)
		}
	}
	output.WriteString("</div>")
	return output.String()
}

func writePageRow(output *strings.Builder, page sdk.Page) {
	output.WriteString(`<a class="widget-row" href="/pages/` + pagePath(page.Slug) + `"><strong>` + html.EscapeString(page.Title) + `</strong><span class="widget-meta">` + html.EscapeString(page.Slug) + `</span></a>`)
}

func pagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}
