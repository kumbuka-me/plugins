package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

const (
	favoritesFeature  = "me.kumbuka.favorites"
	showPinnedFeature = "kumbuka.preference.show-pinned-pages"
	showRecentFeature = "kumbuka.preference.show-recently-viewed"
)

type iconRenderer func(string, int) string

func main() {}

func init() {
	sdk.RegisterWidget("home", renderHomeWidget)
	sdk.RegisterWidget("sidebar", renderSidebarWidget)
}

func renderHomeWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().RecentViewed(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderHome(pages, time.Now(), hostIcon)), nil
}

func renderSidebarWidget(context sdk.WidgetContext) (sdk.Result, error) {
	if !context.Features[showRecentFeature] {
		return sdk.Result{}, nil
	}
	pages, err := sdk.Pages().RecentViewed(8)
	if err != nil {
		return sdk.Result{}, err
	}
	if context.Features[showPinnedFeature] && context.Features[favoritesFeature] {
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
	return sdk.Text(renderSidebar(pages, hostIcon)), nil
}

func renderHome(pages []sdk.Page, now time.Time, icon iconRenderer) string {
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

func renderSidebar(pages []sdk.Page, icon iconRenderer) string {
	var output strings.Builder
	output.WriteString(`<p class="nav-label">Recently viewed</p>`)
	history := icon("history-lucide", 14)
	for _, page := range pages {
		output.WriteString(`<a class="sidebar-shortcut-link" href="/pages/` + pagePath(page.Slug) + `" title="` + html.EscapeString(page.Title) + `">`)
		output.WriteString(history)
		output.WriteString(`<span>` + html.EscapeString(page.Title) + `</span></a>`)
	}
	return output.String()
}

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

func writePageRow(output *strings.Builder, page sdk.Page, now time.Time, icon iconRenderer) {
	output.WriteString(`<a class="page-row" href="/pages/` + pagePath(page.Slug) + `"><span class="doc-icon">`)
	output.WriteString(pageIcon(page, 17, icon))
	output.WriteString(`</span><span><strong>` + html.EscapeString(page.Title) + `</strong><small>` + html.EscapeString(page.Slug))
	if len(page.Tags) != 0 {
		output.WriteString(` · ` + html.EscapeString(strings.Join(page.Tags, ", ")))
	}
	output.WriteString(`</small></span><time>` + relativeTime(page.UpdatedAt, now) + `</time></a>`)
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
