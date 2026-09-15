package main

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
)

type iconRenderer func(string, int) string

func main() {}

func init() { sdk.RegisterWidget("home", renderWidget) }

func renderWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Popular(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderPopular(pages, hostIcon)), nil
}

func renderPopular(pages []sdk.Page, icon iconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>Popular pages</h2></div>`)
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">Popular pages appear after they are viewed.</p>`)
		return output.String()
	}
	for _, page := range pages {
		output.WriteString(`<a class="compact-row" href="/pages/` + pagePath(page.Slug) + `"><span>`)
		output.WriteString(pageIcon(page, 15, icon))
		output.WriteString(`</span><strong>` + html.EscapeString(page.Title) + `</strong><small>` + fmt.Sprintf("%d views", page.ViewCount) + `</small></a>`)
	}
	return output.String()
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
