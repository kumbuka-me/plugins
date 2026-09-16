package main

import (
	"errors"
	"html"
	"net/url"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Wiki Links page-details widget.
func init() { sdk.RegisterWidget("page-details", renderWidget) }

// renderWidget loads incoming and outgoing wiki-link relationships for the current page.
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

// renderLinks renders backlink and outgoing-link sections, including missing destinations.
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
				output.WriteString(`<a class="widget-row" href="/pages/`)
				output.WriteString(widgetui.PagePath(link.TargetSlug))
				output.WriteString(`"><strong>`)
				output.WriteString(html.EscapeString(title))
				output.WriteString(`</strong><span class="widget-meta">`)
				output.WriteString(html.EscapeString(link.TargetSlug))
				output.WriteString(`</span></a>`)
				continue
			}
			output.WriteString(`<a class="widget-row broken" href="/pages/new?slug=`)
			output.WriteString(url.QueryEscape(link.TargetSlug))
			output.WriteString(`"><strong>`)
			output.WriteString(html.EscapeString(link.TargetSlug))
			output.WriteString(`</strong><span class="widget-meta">Missing page</span></a>`)
		}
	}
	output.WriteString("</div>")
	return output.String()
}

// writePageRow appends one resolved backlink row to the widget markup.
func writePageRow(output *strings.Builder, page sdk.Page) {
	output.WriteString(`<a class="widget-row" href="/pages/`)
	output.WriteString(widgetui.PagePath(page.Slug))
	output.WriteString(`"><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><span class="widget-meta">`)
	output.WriteString(html.EscapeString(page.Slug))
	output.WriteString(`</span></a>`)
}
