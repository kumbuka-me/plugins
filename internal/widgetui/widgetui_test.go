package widgetui

import (
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

func TestPagePath(t *testing.T) {
	if got := PagePath("guide/a b"); got != "guide/a%20b" {
		t.Fatalf("PagePath() = %q", got)
	}
}

func TestPageIcon(t *testing.T) {
	render := func(name string, _ int) string { return "[" + name + "]" }
	if got := PageIcon(sdk.Page{Icon: "book-lucide"}, 16, render); got != "[book-lucide]" {
		t.Fatalf("PageIcon() = %q", got)
	}
	if got := PageIcon(sdk.Page{}, 16, render); got != "[file-text-lucide]" {
		t.Fatalf("PageIcon() fallback = %q", got)
	}
}

func TestRelativeTime(t *testing.T) {
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name  string
		value time.Time
		want  string
	}{
		{name: "zero", value: time.Time{}, want: ""},
		{name: "future", value: now.Add(time.Minute), want: "just now"},
		{name: "minutes", value: now.Add(-12 * time.Minute), want: "12m ago"},
		{name: "hours", value: now.Add(-3 * time.Hour), want: "3h ago"},
		{name: "days", value: now.Add(-49 * time.Hour), want: "2d ago"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := RelativeTime(test.value, now); got != test.want {
				t.Fatalf("RelativeTime() = %q, want %q", got, test.want)
			}
		})
	}
}
