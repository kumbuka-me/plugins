package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

func main() {}

func init() { sdk.RegisterWidget("home", renderWidget) }

func renderWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Popular(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderPopular(pages, widgetui.HostIcon)), nil
}

func renderPopular(pages []sdk.Page, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>Popular pages</h2></div>`)
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">Popular pages appear after they are viewed.</p>`)
		return output.String()
	}
	for _, page := range pages {
		output.WriteString(`<a class="compact-row" href="/pages/`)
		output.WriteString(widgetui.PagePath(page.Slug))
		output.WriteString(`"><span>`)
		output.WriteString(widgetui.PageIcon(page, 15, icon))
		output.WriteString(`</span><strong>`)
		output.WriteString(html.EscapeString(page.Title))
		output.WriteString(`</strong><small>`)
		fmt.Fprintf(&output, "%d views", page.ViewCount)
		output.WriteString(`</small></a>`)
	}
	return output.String()
}
