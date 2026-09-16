package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

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
	return sdk.Text(renderHome(pages, widgetui.HostIcon)), nil
}

func renderSidebarWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Favorites(100)
	if err != nil {
		return sdk.Result{}, err
	}
	if len(pages) == 0 {
		return sdk.Result{}, nil
	}
	return sdk.Text(renderSidebar(pages, widgetui.HostIcon)), nil
}

func renderHome(pages []sdk.Page, icon widgetui.IconRenderer) string {
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

func renderSidebar(pages []sdk.Page, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<p class="nav-label">Pinned</p>`)
	star := icon("star-lucide", 14)
	for _, page := range pages {
		output.WriteString(`<a class="sidebar-shortcut-link" href="/pages/`)
		output.WriteString(widgetui.PagePath(page.Slug))
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

func writeCompactRow(output *strings.Builder, page sdk.Page, icon widgetui.IconRenderer) {
	output.WriteString(`<a class="compact-row" href="/pages/`)
	output.WriteString(widgetui.PagePath(page.Slug))
	output.WriteString(`"><span>`)
	output.WriteString(widgetui.PageIcon(page, 15, icon))
	output.WriteString(`</span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	fmt.Fprintf(output, "%d views", page.ViewCount)
	output.WriteString(`</small></a>`)
}
