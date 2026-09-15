package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

const showPinnedFeature = "kumbuka.preference.show-pinned-pages"

type iconRenderer func(string, int) string

func main() {}

func init() {
	sdk.RegisterWidget("home", renderHomeWidget)
	sdk.RegisterWidget("sidebar", renderSidebarWidget)
}

func renderHomeWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Favorites(100)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderHome(pages, hostIcon)), nil
}

func renderSidebarWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if !context.Features[showPinnedFeature] {
		return sdk.Result{}, nil
	}
	pages, err := sdk.Pages().Favorites(100)
	if err != nil {
		return sdk.Result{}, err
	}
	if len(pages) == 0 {
		return sdk.Result{}, nil
	}
	return sdk.Text(renderSidebar(pages, hostIcon)), nil
}

func renderHome(pages []sdk.Page, icon iconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2 class="heading-with-icon">`)
	output.WriteString(icon("star-lucide", 15))
	output.WriteString(`<span>Favorites</span></h2></div>`)
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">Star useful pages to keep them close.</p>`)
		return output.String()
	}
	for _, page := range pages {
		writeCompactRow(&output, page, icon)
	}
	return output.String()
}

func renderSidebar(pages []sdk.Page, icon iconRenderer) string {
	var output strings.Builder
	output.WriteString(`<p class="nav-label">Pinned</p>`)
	star := icon("star-lucide", 14)
	for _, page := range pages {
		output.WriteString(`<a class="sidebar-shortcut-link" href="/pages/`)
		output.WriteString(pagePath(page.Slug))
		output.WriteString(`" title="`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`">`)
		output.WriteString(star)
		output.WriteString(`<span>`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`</span></a>`)
	}
	return output.String()
}

func writeCompactRow(output *strings.Builder, page sdk.Page, icon iconRenderer) {
	output.WriteString(`<a class="compact-row" href="/pages/`)
	output.WriteString(pagePath(page.Slug))
	output.WriteString(`"><span>`)
	output.WriteString(pageIcon(page, 15, icon))
	output.WriteString(`</span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	fmt.Fprintf(output, "%d views", page.ViewCount)
	output.WriteString(`</small></a>`)
}

func pageIcon(page sdk.Page, size int, icon iconRenderer) string {
	if page.Icon != "" {
		if rendered := icon(page.Icon, size); rendered != "" {
			return rendered
		}
	}
	return icon("file-text-lucide", size)
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
