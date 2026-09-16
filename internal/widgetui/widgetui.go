// Package widgetui provides shared presentation helpers for first-party widgets.
package widgetui

import (
	"fmt"
	"html"
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

// WritePageRow appends the standard timestamped page row used by dashboard widgets.
func WritePageRow(output *strings.Builder, page sdk.Page, now time.Time, icon IconRenderer) {
	output.WriteString(`<a class="page-row" href="/pages/`)
	output.WriteString(PagePath(page.Slug))
	output.WriteString(`"><span class="doc-icon">`)
	output.WriteString(PageIcon(page, 17, icon))
	output.WriteString(`</span><span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	output.WriteString(html.EscapeString(page.Slug))
	if len(page.Tags) != 0 {
		output.WriteString(` · `)
		output.WriteString(html.EscapeString(strings.Join(page.Tags, ", ")))
	}
	output.WriteString(`</small></span><time>`)
	output.WriteString(RelativeTime(page.UpdatedAt, now))
	output.WriteString(`</time></a>`)
}

// WriteCompactPageRow appends the standard compact page row with its view count.
func WriteCompactPageRow(output *strings.Builder, page sdk.Page, icon IconRenderer) {
	output.WriteString(`<a class="compact-row" href="/pages/`)
	output.WriteString(PagePath(page.Slug))
	output.WriteString(`"><span>`)
	output.WriteString(PageIcon(page, 15, icon))
	output.WriteString(`</span><strong>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</strong><small>`)
	fmt.Fprintf(output, "%d views", page.ViewCount)
	output.WriteString(`</small></a>`)
}

// WriteSidebarShortcut appends one escaped sidebar shortcut with a pre-rendered icon.
func WriteSidebarShortcut(output *strings.Builder, page sdk.Page, icon string) {
	output.WriteString(`<a class="sidebar-shortcut-link" href="/pages/`)
	output.WriteString(PagePath(page.Slug))
	output.WriteString(`" title="`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`">`)
	output.WriteString(icon)
	output.WriteString(`<span>`)
	output.WriteString(html.EscapeString(page.Title))
	output.WriteString(`</span></a>`)
}
