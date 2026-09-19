package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"
)

const (
	cacheSettingKey = "cache.ttl"
	defaultCacheTTL = time.Hour
	maxCacheTTL     = 24 * time.Hour
	maxCacheEntries = 128
)

// cacheEntry contains one complete validated provider file and its cache metadata.
type cacheEntry struct {
	// SourceFingerprint invalidates the entry when its repository connection changes.
	SourceFingerprint string
	// FetchedAt records when the provider returned this content.
	FetchedAt time.Time
	// LastUsed records the most recent render that referenced this entry.
	LastUsed time.Time
	// Content stores the complete validated external file.
	Content string
}

// fileCacheState protects the shared file cache and administrator invalidation boundary.
type fileCacheState struct {
	// Mutex serializes entry updates and cache invalidation.
	sync.Mutex
	// Entries maps source-and-path fingerprints to complete cached files.
	Entries map[string]cacheEntry
	// RefreshAfter marks fetches at or before this time stale; zero means no manual refresh.
	RefreshAfter time.Time
}

var externalFileCache = fileCacheState{Entries: make(map[string]cacheEntry)}

// loadCacheTTL reads the plugin-owned cache duration and falls back to one hour.
func loadCacheTTL(read settingsReader) time.Duration {
	if read == nil {
		return defaultCacheTTL
	}
	value, err := read(cacheSettingKey)
	if err != nil || !value.Found {
		return defaultCacheTTL
	}
	ttl, err := time.ParseDuration(strings.TrimSpace(string(value.Value)))
	if err != nil || ttl <= 0 || ttl > maxCacheTTL {
		return defaultCacheTTL
	}
	return ttl
}

// cachedFile loads a fresh cached file or lazily refreshes a stale entry on access.
func cachedFile(sourceName string, connection source, path string, ttl time.Duration, now time.Time, fetch httpDoer) (string, error) {
	key := fileCacheKey(sourceName, path)
	fingerprint := sourceFingerprint(connection)
	entry, usable, fresh := readFileCache(key, fingerprint, now, ttl)
	if fresh {
		return entry.Content, nil
	}

	if !allowSourceFetch(sourceName) {
		if usable {
			return entry.Content, nil
		}
		return "", errUnavailable
	}

	content, err := fetchFile(connection, path, fetch)
	if err != nil {
		if usable {
			return entry.Content, nil
		}
		return "", err
	}

	writeFileCache(key, cacheEntry{
		SourceFingerprint: fingerprint,
		FetchedAt:         now.UTC(),
		LastUsed:          now.UTC(),
		Content:           content,
	})
	return content, nil
}

// readFileCache returns one usable entry and reports separately whether it is still fresh.
func readFileCache(key, fingerprint string, now time.Time, ttl time.Duration) (cacheEntry, bool, bool) {
	externalFileCache.Lock()
	defer externalFileCache.Unlock()

	entry, ok := externalFileCache.Entries[key]
	if !ok || entry.SourceFingerprint != fingerprint || !validContent(entry.Content) || entry.FetchedAt.IsZero() {
		return cacheEntry{}, false, false
	}

	entry.LastUsed = now.UTC()
	externalFileCache.Entries[key] = entry
	fresh := cacheEntryFresh(entry, now, ttl, externalFileCache.RefreshAfter)
	return entry, true, fresh
}

// writeFileCache stores one entry and evicts the least recently used item when the cache is full.
func writeFileCache(key string, entry cacheEntry) {
	if key == "" || entry.SourceFingerprint == "" || entry.FetchedAt.IsZero() || !validContent(entry.Content) {
		return
	}

	externalFileCache.Lock()
	defer externalFileCache.Unlock()
	if _, exists := externalFileCache.Entries[key]; !exists && len(externalFileCache.Entries) >= maxCacheEntries {
		evictOldestCacheEntry()
	}
	externalFileCache.Entries[key] = entry
}

// evictOldestCacheEntry removes the least recently used cached file while the cache lock is held.
func evictOldestCacheEntry() {
	oldestKey := ""
	var oldest time.Time
	for key, entry := range externalFileCache.Entries {
		used := entry.LastUsed
		if used.IsZero() {
			used = entry.FetchedAt
		}
		if oldestKey == "" || used.Before(oldest) {
			oldestKey = key
			oldest = used
		}
	}
	if oldestKey != "" {
		delete(externalFileCache.Entries, oldestKey)
	}
}

// cacheEntryFresh reports whether an entry is inside the configured TTL and global refresh boundary.
func cacheEntryFresh(entry cacheEntry, now time.Time, ttl time.Duration, refreshAfter time.Time) bool {
	fetchedAt := entry.FetchedAt.UTC()
	if fetchedAt.After(now.Add(time.Minute)) ||
		(!refreshAfter.IsZero() && !fetchedAt.After(refreshAfter)) {
		return false
	}
	age := now.Sub(fetchedAt)
	return age >= 0 && age < ttl
}

// refreshCache marks every cached file stale so the next visit fetches it again.
func refreshCache() {
	refreshCacheAt(time.Now().UTC())
}

// refreshCacheAt stores one global invalidation boundary for deterministic tests.
func refreshCacheAt(now time.Time) {
	externalFileCache.Lock()
	externalFileCache.RefreshAfter = now.UTC()
	externalFileCache.Unlock()
}

// resetExternalFileCache clears process-local cache state between tests.
func resetExternalFileCache() {
	externalFileCache.Lock()
	externalFileCache.Entries = make(map[string]cacheEntry)
	externalFileCache.RefreshAfter = time.Time{}
	externalFileCache.Unlock()
}

// fileCacheKey creates a stable in-memory key for one configured source and repository path.
func fileCacheKey(sourceName, path string) string {
	sum := sha256.Sum256([]byte(sourceName + "\x00" + path))
	return hex.EncodeToString(sum[:])
}

// sourceFingerprint detects any repository connection change, including credential changes.
func sourceFingerprint(value source) string {
	// source contains only JSON-serializable connection settings, so new settings
	// automatically participate in invalidation without a second field list.
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
