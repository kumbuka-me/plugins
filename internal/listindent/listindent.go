// Package listindent provides shared, validated list indentation presets.
package listindent

import (
	"io"
	"strings"

	sdk "github.com/kumbuka-me/sdk"
	xhtml "golang.org/x/net/html"
)

const Default = "default"

// Reader reads one plugin-owned setting.
type Reader func(string) (sdk.StoredValue, error)

// Load returns the configured preset, falling back safely when no valid value is stored.
func Load(read Reader) string {
	if read == nil {
		return Default
	}
	stored, err := read("layout.indent")
	if err != nil || !stored.Found {
		return Default
	}
	value := strings.ToLower(strings.TrimSpace(string(stored.Value)))
	switch value {
	case "compact", Default, "comfortable", "wide":
		return value
	default:
		return Default
	}
}

// Class returns a namespaced class for one validated indentation preset.
func Class(namespace, preset string) string {
	return "kumbuka-" + namespace + "-indent-" + preset
}

// MarkTags adds className to every matching HTML element while preserving all non-tag bytes.
func MarkTags(source, tag, className string) (string, error) {
	tokenizer := xhtml.NewTokenizer(strings.NewReader(source))
	var output strings.Builder
	output.Grow(len(source) + 64)
	for {
		tokenType := tokenizer.Next()
		if tokenType == xhtml.ErrorToken {
			if tokenizer.Err() == io.EOF {
				return output.String(), nil
			}
			return "", tokenizer.Err()
		}
		if tokenType != xhtml.StartTagToken || tokenizer.Token().Data != tag {
			output.Write(tokenizer.Raw())
			continue
		}
		token := tokenizer.Token()
		addClass(&token, className)
		output.WriteString(token.String())
	}
}

func addClass(token *xhtml.Token, className string) {
	for index := range token.Attr {
		if token.Attr[index].Key != "class" {
			continue
		}
		for _, existing := range strings.Fields(token.Attr[index].Val) {
			if existing == className {
				return
			}
		}
		token.Attr[index].Val = strings.TrimSpace(token.Attr[index].Val + " " + className)
		return
	}
	token.Attr = append(token.Attr, xhtml.Attribute{Key: "class", Val: className})
}
