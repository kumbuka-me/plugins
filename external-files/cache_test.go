package main

import (
	"errors"
	"net/http"
	"strconv"
	"testing"
	"time"

	sdk "github.com/kumbuka-me/sdk"
	"github.com/stretchr/testify/require"
)

// TestCacheTTLDefaultsAndConfiguredValue verifies the manifest default is mirrored safely in plugin code.
func TestCacheTTLDefaultsAndConfiguredValue(t *testing.T) {
	got := loadCacheTTL(nil)
	require.Equal(t, time.Hour, got, "default TTL = %s", got)

	settings := func(key string) (sdk.StoredValue, error) {
		require.Equal(t, cacheSettingKey, key, "unexpected setting %q", key)
		return sdk.StoredValue{Found: true, Value: []byte("6h")}, nil
	}
	got = loadCacheTTL(settings)
	require.Equal(t, 6*time.Hour, got, "configured TTL = %s", got)

	invalid := func(string) (sdk.StoredValue, error) {
		return sdk.StoredValue{Found: true, Value: []byte("forever")}, nil
	}
	got = loadCacheTTL(invalid)
	require.Equal(t, time.Hour, got, "invalid TTL fallback = %s", got)
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
		require.FailNow(t, "fresh cache hit reached provider")
		return sdk.HTTPResponse{}, nil
	})
	require.NoError(t, err, "cached content = %q, %v", content, err)
	require.Equal(t, "cached", content, "cached content = %q, %v", content, err)
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
	require.NoError(t, err, "refreshed content = %q, %v", content, err)
	require.Equal(t, "fresh", content, "refreshed content = %q, %v", content, err)
	entry, usable, fresh := readFileCache(key, sourceFingerprint(connection), now, time.Hour)
	require.True(t, usable, "stored cache entry = %+v usable=%t fresh=%t", entry, usable, fresh)
	require.True(t, fresh, "stored cache entry = %+v usable=%t fresh=%t", entry, usable, fresh)
	require.Equal(t, "fresh", entry.Content, "stored cache entry = %+v usable=%t fresh=%t", entry, usable, fresh)
	require.True(t, entry.FetchedAt.Equal(now), "stored cache entry = %+v usable=%t fresh=%t", entry, usable, fresh)
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
	require.NoError(t, err, "stale fallback = %q, %v", content, err)
	require.Equal(t, "last-known-good", content, "stale fallback = %q, %v", content, err)
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
	require.True(t, usable, "manual refresh state = %+v usable=%t fresh=%t", entry, usable, fresh)
	require.False(t, fresh, "manual refresh state = %+v usable=%t fresh=%t", entry, usable, fresh)
	require.Equal(t, "cached", entry.Content, "manual refresh state = %+v usable=%t fresh=%t", entry, usable, fresh)
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
	require.Equal(t, maxCacheEntries, count, "cache entries = %d, want %d", count, maxCacheEntries)
}

// TestSourceFingerprintChangesWithConnection verifies source edits invalidate the existing cache value.
func TestSourceFingerprintChangesWithConnection(t *testing.T) {
	first := testCacheSource()
	second := first
	second.Ref = "release"
	require.NotEqual(t, sourceFingerprint(second), sourceFingerprint(first), "source revision did not affect cache fingerprint")
}

// testCacheSource returns one valid GitLab connection whose responses are plain text.
func testCacheSource() source {
	return source{Provider: "gitlab", Endpoint: "https://gitlab.com/api/v4", Repository: "team/docs", Ref: "main"}
}
