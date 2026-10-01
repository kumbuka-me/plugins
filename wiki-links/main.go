package main

import (
	"errors"
	"html"
	"strings"

	"github.com/kumbuka-me/plugins/internal/localize"
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
	return sdk.Text(renderLinks(links, localize.For(context.Locale))), nil
}

// renderLinks renders backlink and outgoing-link sections, including missing destinations.
func renderLinks(links sdk.PageLinks, localizer sdk.Localizer) string {
	var output strings.Builder
	output.WriteString("<h2>" + localizer.Text("wiki.referenced_by") + "</h2>")
	output.WriteString(`<div class="widget-list">`)
	if len(links.Backlinks) == 0 {
		output.WriteString(`<p class="muted">` + localizer.Text("wiki.no_backlinks") + `</p>`)
	} else {
		for _, page := range links.Backlinks {
			writePageRow(&output, page)
		}
	}
	output.WriteString("</div>")

	output.WriteString("<h2>" + localizer.Text("wiki.links_from") + "</h2>")
	output.WriteString(`<div class="widget-list">`)
	if len(links.Outgoing) == 0 {
		output.WriteString(`<p class="muted">` + localizer.Text("wiki.no_links") + `</p>`)
	} else {
		for _, link := range links.Outgoing {
			if link.Exists {
				title := link.TargetTitle
				if title == "" {
					title = link.TargetSlug
				}
				output.WriteString(`<a class="widget-row" href="`)
				output.WriteString(html.EscapeString(link.TargetURL))
				output.WriteString(`"><strong>`)
				output.WriteString(html.EscapeString(title))
				output.WriteString(`</strong><span class="widget-meta">`)
				output.WriteString(html.EscapeString(link.TargetSlug))
				output.WriteString(`</span></a>`)
				continue
			}
			output.WriteString(`<a class="widget-row broken" href="`)
			output.WriteString(html.EscapeString(link.CreateURL))
			output.WriteString(`"><strong>`)
			output.WriteString(html.EscapeString(link.TargetSlug))
			output.WriteString(`</strong><span class="widget-meta">` + localizer.Text("wiki.missing") + `</span></a>`)
		}
	}
	output.WriteString("</div>")
	return output.String()
}

// writePageRow appends one resolved backlink row to the widget markup.
func writePageRow(output *strings.Builder, page sdk.Page) {
	output.WriteString(`<a class="widget-row" href="`)
	output.WriteString(html.EscapeString(page.URL))
	output.WriteString(`"><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><span class="widget-meta">`)
	output.WriteString(html.EscapeString(page.Slug))
	output.WriteString(`</span></a>`)
}
