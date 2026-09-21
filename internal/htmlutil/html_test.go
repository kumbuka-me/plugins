package htmlutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFragmentRoundTrip(t *testing.T) {
	root, err := ParseFragment(`<p class="a">text</p>`)
	require.NoError(t, err)
	paragraph := root.FirstChild
	AddClass(paragraph, "b")
	SetAttribute(paragraph, "data-test", "yes")
	require.True(t, HasAttribute(paragraph, "data-test"), "unexpected attributes: %#v", paragraph.Attr)
	require.Equal(t, "a b", Attribute(paragraph, "class"), "unexpected attributes: %#v", paragraph.Attr)
	output, err := RenderChildren(root)
	require.NoError(t, err)
	require.Equal(t, `<p class="a b" data-test="yes">text</p>`, output, "RenderChildren() = %q", output)
}
