package main

import (
	"errors"
	"html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

const relatedLimit = 6

// main provides the WASI plugin entry point.
func main() {}

// init registers the Related Pages page-details widget.
func init() { sdk.RegisterWidget("page-details", renderWidget) }

// renderWidget finds visible pages that share the current page's first tag.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if context.Page == nil {
		return sdk.Result{}, errors.New("related pages requires a page")
	}
	if len(context.Page.Tags) == 0 {
		return sdk.Text(renderRelated(nil)), nil
	}

	pages, err := sdk.Pages().Search(sdk.PageQuery{Query: "tag:" + context.Page.Tags[0], Limit: relatedLimit})
	if err != nil {
		return sdk.Result{}, err
	}
	filtered := pages[:0]
	for _, page := range pages {
		if page.Slug == context.Page.Slug {
			continue
		}
		filtered = append(filtered, page)
		if len(filtered) == relatedLimit {
			break
		}
	}
	return sdk.Text(renderRelated(filtered)), nil
}

// renderRelated renders related-page links or an empty state.
func renderRelated(pages []sdk.Page) string {
	var output strings.Builder
	output.WriteString("<h2>Related pages</h2>")
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">No related pages yet.</p>`)
		return output.String()
	}
	output.WriteString(`<div class="widget-list">`)
	for _, page := range pages {
		output.WriteString(`<a class="widget-row" href="/pages/`)
		output.WriteString(widgetui.PagePath(page.Slug))
		output.WriteString(`"><strong>`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`</strong><span class="widget-meta">`)
		output.WriteString(html.EscapeString(page.Slug))
		output.WriteString(`</span></a>`)
	}
	output.WriteString("</div>")
	return output.String()
}
