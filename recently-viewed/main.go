package main

import (
	"html"
	"strings"
	"time"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

const favoritesFeature = "me.kumbuka.favorites"

// main provides the WASI plugin entry point.
func main() {}

// init registers the Recently Viewed home and sidebar widgets.
func init() {
	sdk.RegisterWidget("home", renderHomeWidget)
	sdk.RegisterWidget("sidebar", renderSidebarWidget)
}

// renderHomeWidget loads recently viewed pages for the home dashboard surface.
func renderHomeWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().RecentViewed(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderHome(pages, time.Now(), widgetui.HostIcon)), nil
}

// renderSidebarWidget loads sidebar history while excluding pinned favorites when needed.
func renderSidebarWidget(context sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().RecentViewed(8)
	if err != nil {
		return sdk.Result{}, err
	}
	if context.Features[favoritesFeature] {
		favorites, favoriteErr := sdk.Pages().Favorites(100)
		if favoriteErr != nil {
			return sdk.Result{}, favoriteErr
		}
		pages = withoutFavorites(pages, favorites, 5)
	} else if len(pages) > 5 {
		pages = pages[:5]
	}
	if len(pages) == 0 {
		return sdk.Result{}, nil
	}
	return sdk.Text(renderSidebar(pages, widgetui.HostIcon)), nil
}

// renderHome renders the Recently Viewed dashboard panel.
func renderHome(pages []sdk.Page, now time.Time, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>Recently viewed</h2></div>`)
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">Pages you open will appear here.</p>`)
		return output.String()
	}
	for _, page := range pages {
		writePageRow(&output, page, now, icon)
	}
	return output.String()
}

// renderSidebar renders recently viewed shortcuts for the sidebar.
func renderSidebar(pages []sdk.Page, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<p class="nav-label">Recently viewed</p>`)
	history := icon("history-lucide", 14)
	for _, page := range pages {
		output.WriteString(`<a class="sidebar-shortcut-link" href="/pages/`)
		output.WriteString(widgetui.PagePath(page.Slug))
		output.WriteString(`" title="`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`">`)
		output.WriteString(history)
		output.WriteString(`<span>`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`</span></a>`)
	}
	return output.String()
}

// withoutFavorites removes favorite pages while preserving recent-view order and the requested limit.
func withoutFavorites(pages, favorites []sdk.Page, limit int) []sdk.Page {
	favorite := make(map[string]bool, len(favorites))
	for _, page := range favorites {
		favorite[page.Slug] = true
	}
	result := make([]sdk.Page, 0, min(len(pages), limit))
	for _, page := range pages {
		if favorite[page.Slug] {
			continue
		}
		result = append(result, page)
		if len(result) == limit {
			break
		}
	}
	return result
}

// writePageRow appends one recently viewed page row to the home widget.
func writePageRow(output *strings.Builder, page sdk.Page, now time.Time, icon widgetui.IconRenderer) {
	output.WriteString(`<a class="page-row" href="/pages/`)
	output.WriteString(widgetui.PagePath(page.Slug))
	output.WriteString(`"><span class="doc-icon">`)
	output.WriteString(widgetui.PageIcon(page, 17, icon))
	output.WriteString(`</span><span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	output.WriteString(html.EscapeString(page.Slug))
	if len(page.Tags) != 0 {
		output.WriteString(` · `)
		output.WriteString(html.EscapeString(strings.Join(page.Tags, ", ")))
	}
	output.WriteString(`</small></span><time>`)
	output.WriteString(widgetui.RelativeTime(page.UpdatedAt, now))
	output.WriteString(`</time></a>`)
}
