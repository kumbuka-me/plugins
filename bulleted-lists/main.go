package main

import (
	"github.com/kumbuka-me/plugins/internal/listindent"
	sdk "github.com/kumbuka-me/sdk"
)

func main() {}

func init() { sdk.RegisterModule("indent", transform) }

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
