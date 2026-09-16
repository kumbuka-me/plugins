package markdownblock

import (
	"testing"

	sdk "github.com/kumbuka-me/sdk"
)

func TestAppendTextCoalescesLiteralParts(t *testing.T) {
	parts := []sdk.RenderPart{{Text: "a"}}
	AppendText(&parts, "b")
	if len(parts) != 1 || parts[0].Text != "ab" {
		t.Fatalf("unexpected parts: %#v", parts)
	}
}

func TestParseQuotedTitle(t *testing.T) {
	if title, ok := ParseQuotedTitle(`"Hello"`); !ok || title != "Hello" {
		t.Fatalf("ParseQuotedTitle() = %q, %t", title, ok)
	}
	if _, ok := ParseQuotedTitle(`""`); ok {
		t.Fatal("empty title accepted")
	}
}

func TestIndentedBody(t *testing.T) {
	body, next := IndentedBody([]string{"header", "    one", "", "\ttwo", "stop"}, 1)
	if next != 4 || len(body) != 3 || body[0] != "one" || body[1] != "" || body[2] != "two" {
		t.Fatalf("IndentedBody() = %#v, %d", body, next)
	}
}
