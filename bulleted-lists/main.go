package main

import (
	"github.com/kumbuka-me/plugins/internal/listindent"
	sdk "github.com/kumbuka-me/sdk"
)

// main provides the entry point required by the plugin executable.
func main() {}

// init registers the plugin's render and command handlers with the SDK.
func init() { sdk.RegisterModule("indent", transform) }

// transform adds configured indentation to unordered lists.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "indent" || request.Stage != "postprocess" {
		return sdk.RenderResult{Error: "unsupported bulleted-list render request"}
	}
	preset := listindent.Load(sdk.Settings().Get)
	output, err := listindent.MarkTags(request.Source, "ul", listindent.Class("bulleted-list", preset))
	if err != nil {
		return sdk.RenderResult{Error: err.Error()}
	}
	return sdk.Text(output)
}
