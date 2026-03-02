package calendar

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultCacheTTL = 15 * time.Minute

type cacheEntry struct {
	FetchedAt time.Time `json:"fetched_at"`
	Date      string    `json:"date"` // "2006-01-02": invalidates across midnight
	Events    []*Event  `json:"events"`
}

// readCache returns cached events for a feed URL if the entry exists, is for
// today, and was fetched within ttl. Returns (nil, false) on any miss.
func readCache(feedURL, today string, ttl time.Duration) ([]*Event, bool) {
	path, err := cachePath(feedURL)
	if err != nil {
		return nil, false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}

	if entry.Date != today {
		return nil, false // stale date
	}
	if time.Since(entry.FetchedAt) > ttl {
		return nil, false // TTL expired
	}

	return entry.Events, true
}

// writeCache persists events for a feed URL to disk.
func writeCache(feedURL, today string, events []*Event) {
	path, err := cachePath(feedURL)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return
	}

	entry := cacheEntry{
		FetchedAt: time.Now(),
		Date:      today,
		Events:    events,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	// Best-effort write; ignore errors so a cache failure never breaks fl today.
	_ = os.WriteFile(path, data, 0600)
}

func cachePath(feedURL string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("%x", sha256.Sum256([]byte(feedURL)))[:16]
	return filepath.Join(home, ".fl", "cache", "ical", key+".json"), nil
}
