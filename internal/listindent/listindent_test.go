package listindent

import (
	"strings"
	"testing"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

func TestLoadAndMarkTags(t *testing.T) {
	preset := Load(func(string) (sdk.StoredValue, error) {
		return sdk.StoredValue{Found: true, Value: []byte("Comfortable")}, nil
	})
	require.Equal(t, "comfortable", preset)

	got, err := MarkTags(`<ul class="existing"><li>One</li><li><ul><li>Two</li></ul></li></ul><ol><li>Three</li></ol>`, "ul", Class("bulleted-list", preset))
	require.NoError(t, err)
	require.Equal(t, 2, strings.Count(got, "kumbuka-bulleted-list-indent-comfortable"))
	require.Contains(t, got, `<ol><li>Three</li></ol>`)
}

func TestMarkTagsPreservesTagsAndAttributes(t *testing.T) {
	source := `<section data-label="A &amp; B"><ul id="items" class="existing"><li>One &amp; two</li></ul></section>`
	got, err := MarkTags(source, "ul", "indent")
	require.NoError(t, err)
	require.Equal(t, `<section data-label="A &amp; B"><ul id="items" class="existing indent"><li>One &amp; two</li></ul></section>`, got)

	again, err := MarkTags(got, "ul", "indent")
	require.NoError(t, err)
	require.Equal(t, got, again)
}
