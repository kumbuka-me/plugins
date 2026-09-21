package widgetui

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPagePath(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "guide/a%20b", PagePath("guide/a b"))
	assert.Equal(t, "guide/%22%3C%3E%23%3F%25", PagePath("guide/\"<>#?%"))
	assert.Empty(t, PagePath(""))
	assert.Equal(t, "/guide//", PagePath("/guide//"))
}

func TestPageIcon(t *testing.T) {
	t.Parallel()
	calls := 0
	got := PageIcon(sdk.Page{Icon: "book-lucide"}, 16, func(name string, size int) string {
		calls++
		assert.Equal(t, "book-lucide", name)
		assert.Equal(t, 16, size)
		return "<svg/>"
	})
	assert.Equal(t, "<svg/>", got)
	assert.Equal(t, 1, calls)
}

func TestPageIconWithoutCustomIcon(t *testing.T) {
	t.Parallel()
	calls := 0
	got := PageIcon(sdk.Page{}, 17, func(name string, size int) string {
		calls++
		assert.Equal(t, "file-text-lucide", name)
		assert.Equal(t, 17, size)
		return "default"
	})
	assert.Equal(t, "default", got)
	assert.Equal(t, 1, calls)
}

func TestPageIconFallsBackWhenCustomIconUnavailable(t *testing.T) {
	t.Parallel()
	var calls []string
	got := PageIcon(sdk.Page{Icon: "missing"}, 15, func(name string, size int) string {
		calls = append(calls, name)
		assert.Equal(t, 15, size)
		if name == "missing" {
			return ""
		}
		return "default"
	})
	assert.Equal(t, "default", got)
	assert.Equal(t, []string{"missing", "file-text-lucide"}, calls)
}

func TestRelativeTime(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	t.Run("zero", func(t *testing.T) { t.Parallel(); assert.Empty(t, RelativeTime(time.Time{}, now)) })
	t.Run("future", func(t *testing.T) { t.Parallel(); assert.Equal(t, "just now", RelativeTime(now.Add(time.Minute), now)) })
	t.Run("below minute", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "just now", RelativeTime(now.Add(-time.Minute+time.Nanosecond), now))
	})
	t.Run("minute boundary", func(t *testing.T) { t.Parallel(); assert.Equal(t, "1m ago", RelativeTime(now.Add(-time.Minute), now)) })
	t.Run("below hour", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "59m ago", RelativeTime(now.Add(-time.Hour+time.Nanosecond), now))
	})
	t.Run("hour boundary", func(t *testing.T) { t.Parallel(); assert.Equal(t, "1h ago", RelativeTime(now.Add(-time.Hour), now)) })
	t.Run("below day", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "23h ago", RelativeTime(now.Add(-24*time.Hour+time.Nanosecond), now))
	})
	t.Run("day boundary", func(t *testing.T) { t.Parallel(); assert.Equal(t, "1d ago", RelativeTime(now.Add(-24*time.Hour), now)) })
	t.Run("partial days", func(t *testing.T) { t.Parallel(); assert.Equal(t, "2d ago", RelativeTime(now.Add(-49*time.Hour), now)) })
}

func TestWritePageRow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 16, 12, 0, 0, 0, time.UTC)
	var output strings.Builder
	output.WriteString("prefix")
	WritePageRow(&output, sdk.Page{Slug: "guide/a b", Title: "<Title>&", Tags: []string{"<tag>", "a&b"}, UpdatedAt: now.Add(-time.Hour)}, now, func(name string, size int) string {
		require.Equal(t, "file-text-lucide", name)
		assert.Equal(t, 17, size)
		return "<svg/>"
	})
	assert.Equal(t, `prefix<a class="page-row" href="/pages/guide/a%20b"><span class="doc-icon"><svg/></span><span><strong>&lt;Title&gt;&amp;</strong><small>guide/a b · &lt;tag&gt;, a&amp;b</small></span><time>1h ago</time></a>`, output.String())
}

func TestWritePageRowWithoutTagsOrTimestamp(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	WritePageRow(&output, sdk.Page{Slug: "a&b", Title: "Page"}, time.Time{}, func(string, int) string { return "" })
	assert.Equal(t, `<a class="page-row" href="/pages/a&b"><span class="doc-icon"></span><span><strong>Page</strong><small>a&amp;b</small></span><time></time></a>`, output.String())
}

func TestWriteCompactPageRow(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	WriteCompactPageRow(&output, sdk.Page{Slug: "a b", Title: "<Page>", ViewCount: 42}, func(name string, size int) string {
		require.Equal(t, "file-text-lucide", name)
		assert.Equal(t, 15, size)
		return "<svg/>"
	})
	assert.Equal(t, `<a class="compact-row" href="/pages/a%20b"><span><svg/></span><strong>&lt;Page&gt;</strong><small>42 views</small></a>`, output.String())
}

func TestWriteSidebarShortcut(t *testing.T) {
	t.Parallel()
	var output strings.Builder
	WriteSidebarShortcut(&output, sdk.Page{Slug: "a b", Title: `"<Page>&`}, "<svg/>")
	assert.Equal(t, `<a class="sidebar-shortcut-link" href="/pages/a%20b" title="&#34;&lt;Page&gt;&amp;"><svg/><span>&#34;&lt;Page&gt;&amp;</span></a>`, output.String())
}
