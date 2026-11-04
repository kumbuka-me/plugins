package main

import (
	"bytes"
	"regexp"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	sdk "github.com/kumbuka-me/sdk"
)

var (
	highlightFormatter = chromahtml.New(chromahtml.WithClasses(true))
	highlightStyle     = styles.Get("github-dark")
	highlightLanguages = func() map[string]string {
		names := lexers.Names(true)
		identifier := regexp.MustCompile(`[^a-z0-9+#]+`)
		languages := make(map[string]string, len(names))
		for _, name := range names {
			languages[strings.ToLower(name)] = name
			languages[strings.Trim(identifier.ReplaceAllString(strings.ToLower(name), "_"), "_")] = name
		}
		languages["rpm_spec"] = "RPMSpec"
		languages["yml"] = "YAML"
		return languages
	}()
)

// main provides the WASI plugin entry point.
func main() {}

// init validates the configured Chroma style and registers the code highlighter.
func init() {
	if highlightStyle == nil {
		panic("syntax highlighting style is unavailable")
	}
	sdk.RegisterModule("chroma", transform)
}

// transform highlights an explicitly labeled fenced code block with Chroma without language auto-detection.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "chroma" || request.Stage != "highlight" {
		return sdk.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	language := strings.ToLower(strings.TrimSpace(request.Language))
	if language == "" {
		return sdk.RenderResult{}
	}
	lexerName, ok := highlightLanguages[language]
	if !ok {
		return sdk.RenderResult{}
	}

	// Get selects only the explicitly supplied Chroma name or alias. Do not use
	// Match or Analyse here: fenced code is never language-detected from content.
	lexer := lexers.Get(lexerName)
	if lexer == nil {
		return sdk.RenderResult{}
	}

	iterator, err := lexer.Tokenise(nil, request.Source)
	if err != nil {
		return sdk.RenderResult{Error: "tokenize code: " + err.Error()}
	}

	var output bytes.Buffer
	if err := highlightFormatter.Format(&output, highlightStyle, iterator); err != nil {
		return sdk.RenderResult{Error: "format highlighted code: " + err.Error()}
	}

	return sdk.RenderResult{Matched: true, Parts: []sdk.RenderPart{{Text: output.String()}}}
}
