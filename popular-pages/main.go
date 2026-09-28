package main

import (
	"strings"

	"github.com/kumbuka-me/plugins/internal/localize"
	"github.com/kumbuka-me/plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Popular Pages home widget.
func init() { sdk.RegisterWidget("home", renderWidget) }

// renderWidget loads the most-viewed pages for the home widget.
func renderWidget(context sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Popular(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderPopular(pages, widgetui.HostIcon, localize.For(context.Locale))), nil
}

// renderPopular renders the Popular Pages dashboard panel.
func renderPopular(pages []sdk.Page, icon widgetui.IconRenderer, localizer sdk.Localizer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>` + localizer.Text("popular.title") + `</h2></div>`)
	if len(pages) == 0 {
		output.WriteString(`<p class="muted">` + localizer.Text("popular.empty") + `</p>`)
		return output.String()
	}
	for _, page := range pages {
		widgetui.WriteCompactPageRow(&output, page, icon, localizer)
	}
	return output.String()
}
