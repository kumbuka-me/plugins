package htmlutil

import "testing"

func TestFragmentRoundTrip(t *testing.T) {
	root, err := ParseFragment(`<p class="a">text</p>`)
	if err != nil {
		t.Fatal(err)
	}
	paragraph := root.FirstChild
	AddClass(paragraph, "b")
	SetAttribute(paragraph, "data-test", "yes")
	if !HasAttribute(paragraph, "data-test") || Attribute(paragraph, "class") != "a b" {
		t.Fatalf("unexpected attributes: %#v", paragraph.Attr)
	}
	output, err := RenderChildren(root)
	if err != nil {
		t.Fatal(err)
	}
	if output != `<p class="a b" data-test="yes">text</p>` {
		t.Fatalf("RenderChildren() = %q", output)
	}
}
