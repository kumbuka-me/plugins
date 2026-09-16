// Package widgetui provides shared presentation helpers for first-party widgets.
package widgetui

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

// IconRenderer renders one host icon at the requested pixel size.
type IconRenderer func(name string, size int) string

// HostIcon renders an icon through the SDK and returns an empty string on host errors.
func HostIcon(name string, size int) string {
	rendered, err := sdk.Icon(name, size)
	if err != nil {
		return ""
	}
	return rendered
}

// PageIcon renders a page-specific icon and falls back to the default document icon.
func PageIcon(page sdk.Page, size int, render IconRenderer) string {
	if page.Icon != "" {
		if rendered := render(page.Icon, size); rendered != "" {
			return rendered
		}
	}
	return render("file-text-lucide", size)
}

// PagePath escapes each canonical page slug segment for use in application URLs.
func PagePath(slug string) string {
	parts := strings.Split(slug, "/")
	for index := range parts {
		parts[index] = url.PathEscape(parts[index])
	}
	return strings.Join(parts, "/")
}

// RelativeTime formats a timestamp as the compact age used by dashboard widgets.
func RelativeTime(value, now time.Time) string {
	if value.IsZero() {
		return ""
	}

	delta := now.Sub(value)
	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta/time.Minute))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta/time.Hour))
	default:
		return fmt.Sprintf("%dd ago", int(delta/(24*time.Hour)))
	}
}
