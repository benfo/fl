package calendar

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/apognu/gocal"
	"github.com/benfourie/fl/internal/config"
)

// ForceRefresh bypasses the iCal cache when set to true.
// Set by cmd/today.go when the --refresh flag is passed.
var ForceRefresh bool

// icalTodayEvents fetches all configured iCal feeds in parallel, returning
// events that fall on today. Results are served from cache when fresh.
func icalTodayEvents() ([]*Event, error) {
	feeds, err := config.ICalFeeds()
	if err != nil {
		return nil, err
	}
	if len(feeds) == 0 {
		return nil, nil
	}

	now := time.Now()
	today := now.Format("2006-01-02")
	dayStart := startOfDay(now)
	dayEnd := endOfDay(now)

	type result struct {
		events []*Event
		err    string
	}

	results := make([]result, len(feeds))
	var wg sync.WaitGroup

	for i, feed := range feeds {
		wg.Add(1)
		go func(idx int, f config.ICalFeed) {
			defer wg.Done()
			events, err := fetchFeed(f, today, dayStart, dayEnd)
			if err != nil {
				results[idx] = result{err: fmt.Sprintf("%s: %s", f.Name, err)}
			} else {
				results[idx] = result{events: events}
			}
		}(i, feed)
	}

	wg.Wait()

	var events []*Event
	var errs []string
	for _, r := range results {
		if r.err != "" {
			errs = append(errs, r.err)
		} else {
			events = append(events, r.events...)
		}
	}

	if len(errs) > 0 && len(events) == 0 {
		return nil, fmt.Errorf("ical errors: %s", strings.Join(errs, "; "))
	}
	return events, nil
}

func fetchFeed(feed config.ICalFeed, today string, dayStart, dayEnd time.Time) ([]*Event, error) {
	if !ForceRefresh {
		if cached, ok := readCache(feed.URL, today, defaultCacheTTL); ok {
			return cached, nil
		}
	}

	body, err := download(feed.URL)
	if err != nil {
		return nil, err
	}

	events, err := parseFeed(feed.Name, body, dayStart, dayEnd)
	if err != nil {
		return nil, err
	}

	writeCache(feed.URL, today, events)
	return events, nil
}

// download fetches the raw iCal data. Handles webcal:// by rewriting to https://.
func download(rawURL string) (string, error) {
	u := strings.Replace(rawURL, "webcal://", "https://", 1)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return "", fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	return string(data), nil
}

// parseFeed parses iCal data and returns events that overlap with [dayStart, dayEnd].
// gocal expands RRULE recurring events within the given window automatically.
func parseFeed(feedName, body string, dayStart, dayEnd time.Time) ([]*Event, error) {
	c := gocal.NewParser(strings.NewReader(body))
	c.Start = &dayStart
	c.End = &dayEnd

	if err := c.Parse(); err != nil {
		return nil, fmt.Errorf("parsing iCal: %w", err)
	}

	events := make([]*Event, 0, len(c.Events))
	for _, e := range c.Events {
		ev := mapEvent(e, feedName)
		if ev != nil {
			events = append(events, ev)
		}
	}
	return events, nil
}

func mapEvent(e gocal.Event, feedName string) *Event {
	if e.Start == nil {
		return nil
	}

	// gocal returns VALUE=DATE (all-day) events as midnight UTC with no time
	// component. Detect this before converting to local time.
	allDay := isAllDay(*e.Start)

	// Convert to local time so display is always in the user's timezone,
	// regardless of whether the feed used UTC, a TZID, or floating times.
	start := e.Start.In(time.Local)
	end := start.Add(time.Hour)
	if e.End != nil {
		end = e.End.In(time.Local)
	}

	title := e.Summary
	if title == "" {
		title = "(no title)"
	}

	return &Event{
		Title:    title,
		Start:    start,
		End:      end,
		AllDay:   allDay,
		Location: e.Location,
		Provider: feedName,
	}
}

// isAllDay reports whether t looks like a VALUE=DATE event.
// gocal parses DATE-only values as midnight UTC with no offset.
func isAllDay(t time.Time) bool {
	return t.Location() == time.UTC &&
		t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}
