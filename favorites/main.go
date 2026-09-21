package main

import (
	"strings"

	"github.com/kumbuka-me/plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Favorites home and sidebar widgets.
func init() {
	sdk.RegisterWidget("home", renderHomeWidget)
	sdk.RegisterWidget("sidebar", renderSidebarWidget)
}

// renderHomeWidget loads favorite pages for the home dashboard surface.
func renderHomeWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Favorites(100)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderHome(pages, widgetui.HostIcon)), nil
}

// renderSidebarWidget loads favorite pages for the sidebar surface.
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

// renderHome renders the Favorites dashboard panel.
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
		widgetui.WriteCompactPageRow(&output, page, icon)
	}
	return output.String()
}

// renderSidebar renders pinned favorite links for the sidebar.
func renderSidebar(pages []sdk.Page, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<p class="nav-label">Pinned</p>`)
	star := icon("star-lucide", 14)
	for _, page := range pages {
		widgetui.WriteSidebarShortcut(&output, page, star)
	}
	return output.String()
}
