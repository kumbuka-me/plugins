package main

import (
	"bytes"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	sdk "github.com/kumbuka-me/sdk"
)

var (
	highlightFormatter = chromahtml.New(chromahtml.WithClasses(true))
	highlightStyle     = styles.Get("github-dark")
	highlightLanguages = func() map[string]struct{} {
		names := lexers.Names(true)
		languages := make(map[string]struct{}, len(names))
		for _, name := range names {
			languages[strings.ToLower(name)] = struct{}{}
		}
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

// transform highlights one fenced code block using its explicit Markdown
// fence language. Chroma's global registry provides the complete maintained
// lexer set; Kumbuka does not auto-detect a language from the source text.
func transform(request sdk.RenderRequest) sdk.RenderResult {
	if request.Module != "chroma" || request.Stage != "highlight" {
		return sdk.RenderResult{Error: "unsupported syntax-highlighting request"}
	}

	language := strings.ToLower(strings.TrimSpace(request.Language))
	if language == "" {
		return sdk.RenderResult{}
	}
	if _, ok := highlightLanguages[language]; !ok {
		return sdk.RenderResult{}
	}

	// Get selects only the explicitly supplied Chroma name or alias. Do not use
	// Match or Analyse here: fenced code is never language-detected from content.
	lexer := lexers.Get(language)
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
