package main

import (
	"strings"
	"time"

	"github.com/kumbuka-me/kumbuka-plugins/internal/widgetui"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the WASI plugin entry point.
func main() {}

// init registers the Recent Changes home widget.
func init() { sdk.RegisterWidget("home", renderWidget) }

// renderWidget loads the newest visible pages for the home widget.
func renderWidget(sdk.WidgetContext) (sdk.Result, error) {
	pages, err := sdk.Pages().Recent(8)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.Text(renderRecent(pages, time.Now(), widgetui.HostIcon)), nil
}

// renderRecent renders the Recent Changes dashboard panel.
func renderRecent(pages []sdk.Page, now time.Time, icon widgetui.IconRenderer) string {
	var output strings.Builder
	output.WriteString(`<div class="panel-title"><h2>Recent changes</h2><a href="/search">View all</a></div>`)
	if len(pages) == 0 {
		output.WriteString(`<div class="empty"><strong>Your Kumbuka knowledge base is ready.</strong><p>Create the first page and start linking knowledge together.</p></div>`)
		return output.String()
	}
	for _, page := range pages {
		widgetui.WritePageRow(&output, page, now, icon)
	}
	return output.String()
}
