package main

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
)

// TestCacheTTLDefaultsAndConfiguredValue verifies the manifest default is mirrored safely in plugin code.
func TestCacheTTLDefaultsAndConfiguredValue(t *testing.T) {
	if got := loadCacheTTL(nil); got != time.Hour {
		t.Fatalf("default TTL = %s", got)
	}

	settings := func(key string) (sdk.StoredValue, error) {
		if key != cacheSettingKey {
			t.Fatalf("unexpected setting %q", key)
		}
		return sdk.StoredValue{Found: true, Value: []byte("6h")}, nil
	}
	if got := loadCacheTTL(settings); got != 6*time.Hour {
		t.Fatalf("configured TTL = %s", got)
	}

	invalid := func(string) (sdk.StoredValue, error) {
		return sdk.StoredValue{Found: true, Value: []byte("forever")}, nil
	}
	if got := loadCacheTTL(invalid); got != time.Hour {
		t.Fatalf("invalid TTL fallback = %s", got)
	}
}

// TestCachedFileUsesFreshEntry verifies a cache hit avoids provider network access.
func TestCachedFileUsesFreshEntry(t *testing.T) {
	resetExternalFileCache()
	t.Cleanup(resetExternalFileCache)
	now := time.Date(2026, 9, 18, 16, 0, 0, 0, time.UTC)
	connection := testCacheSource()
	key := fileCacheKey("docs", "README.md")
	writeFileCache(key, cacheEntry{
		SourceFingerprint: sourceFingerprint(connection),
		FetchedAt:         now.Add(-30 * time.Minute),
		LastUsed:          now.Add(-30 * time.Minute),
		Content:           "cached",
	})

	content, err := cachedFile("docs", connection, "README.md", time.Hour, now, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		t.Fatal("fresh cache hit reached provider")
		return sdk.HTTPResponse{}, nil
	})
	if err != nil || content != "cached" {
		t.Fatalf("cached content = %q, %v", content, err)
	}
}

// TestCachedFileRefreshesExpiredEntry verifies the first visit after TTL expiry replaces cached content.
func TestCachedFileRefreshesExpiredEntry(t *testing.T) {
	resetExternalFileCache()
	t.Cleanup(resetExternalFileCache)
	now := time.Date(2026, 9, 18, 16, 0, 0, 0, time.UTC)
	connection := testCacheSource()
	key := fileCacheKey("expired", "README.md")
	writeFileCache(key, cacheEntry{
		SourceFingerprint: sourceFingerprint(connection),
		FetchedAt:         now.Add(-2 * time.Hour),
		LastUsed:          now.Add(-2 * time.Hour),
		Content:           "stale",
	})

	content, err := cachedFile("expired", connection, "README.md", time.Hour, now, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		return sdk.HTTPResponse{StatusCode: http.StatusOK, Body: []byte("fresh")}, nil
	})
	if err != nil || content != "fresh" {
		t.Fatalf("refreshed content = %q, %v", content, err)
	}
	entry, usable, fresh := readFileCache(key, sourceFingerprint(connection), now, time.Hour)
	if !usable || !fresh || entry.Content != "fresh" || !entry.FetchedAt.Equal(now) {
		t.Fatalf("stored cache entry = %+v usable=%t fresh=%t", entry, usable, fresh)
	}
}

// TestCachedFileFallsBackToStaleContent verifies provider failures do not discard the last valid cached copy.
func TestCachedFileFallsBackToStaleContent(t *testing.T) {
	resetExternalFileCache()
	t.Cleanup(resetExternalFileCache)
	now := time.Date(2026, 9, 18, 16, 0, 0, 0, time.UTC)
	connection := testCacheSource()
	writeFileCache(fileCacheKey("fallback", "README.md"), cacheEntry{
		SourceFingerprint: sourceFingerprint(connection),
		FetchedAt:         now.Add(-2 * time.Hour),
		LastUsed:          now.Add(-2 * time.Hour),
		Content:           "last-known-good",
	})

	content, err := cachedFile("fallback", connection, "README.md", time.Hour, now, func(sdk.HTTPRequest) (sdk.HTTPResponse, error) {
		return sdk.HTTPResponse{}, errors.New("provider unavailable")
	})
	if err != nil || content != "last-known-good" {
		t.Fatalf("stale fallback = %q, %v", content, err)
	}
}

// TestRefreshCacheMarksFreshEntriesStale verifies the admin action invalidates all entries without deleting fallback content.
func TestRefreshCacheMarksFreshEntriesStale(t *testing.T) {
	resetExternalFileCache()
	t.Cleanup(resetExternalFileCache)
	fetchedAt := time.Date(2026, 9, 18, 15, 30, 0, 0, time.UTC)
	refreshAt := fetchedAt.Add(10 * time.Minute)
	connection := testCacheSource()
	key := fileCacheKey("docs", "README.md")
	writeFileCache(key, cacheEntry{
		SourceFingerprint: sourceFingerprint(connection),
		FetchedAt:         fetchedAt,
		LastUsed:          fetchedAt,
		Content:           "cached",
	})

	refreshCacheAt(refreshAt)
	entry, usable, fresh := readFileCache(key, sourceFingerprint(connection), refreshAt.Add(time.Minute), time.Hour)
	if !usable || fresh || entry.Content != "cached" {
		t.Fatalf("manual refresh state = %+v usable=%t fresh=%t", entry, usable, fresh)
	}
}

// TestFileCacheEvictsLeastRecentlyUsedEntry verifies memory use stays bounded.
func TestFileCacheEvictsLeastRecentlyUsedEntry(t *testing.T) {
	resetExternalFileCache()
	t.Cleanup(resetExternalFileCache)
	base := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	for index := 0; index < maxCacheEntries+1; index++ {
		key := fileCacheKey("source", strconv.Itoa(index))
		writeFileCache(key, cacheEntry{
			SourceFingerprint: "fingerprint",
			FetchedAt:         base.Add(time.Duration(index) * time.Second),
			LastUsed:          base.Add(time.Duration(index) * time.Second),
			Content:           "content",
		})
	}
	externalFileCache.Lock()
	count := len(externalFileCache.Entries)
	externalFileCache.Unlock()
	if count != maxCacheEntries {
		t.Fatalf("cache entries = %d, want %d", count, maxCacheEntries)
	}
}

// TestSourceFingerprintChangesWithConnection verifies source edits invalidate the existing cache value.
func TestSourceFingerprintChangesWithConnection(t *testing.T) {
	first := testCacheSource()
	second := first
	second.Ref = "release"
	if sourceFingerprint(first) == sourceFingerprint(second) {
		t.Fatal("source revision did not affect cache fingerprint")
	}
}

// testCacheSource returns one valid GitLab connection whose responses are plain text.
func testCacheSource() source {
	return source{Provider: "gitlab", Endpoint: "https://gitlab.com/api/v4", Repository: "team/docs", Ref: "main"}
}
