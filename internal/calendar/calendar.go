package calendar

import (
	"time"
)

// Event represents a single calendar event.
type Event struct {
	Title    string
	Start    time.Time
	End      time.Time
	Location string
	Provider string // "google" | "outlook" | <ical feed name>
}

// TodayEvents fetches today's events from all configured calendar providers.
// Errors from individual providers are collected but non-fatal.
func TodayEvents() ([]*Event, error) {
	var events []*Event

	googleEvents, err := googleTodayEvents()
	if err == nil {
		events = append(events, googleEvents...)
	}

	outlookEvents, err := outlookTodayEvents()
	if err == nil {
		events = append(events, outlookEvents...)
	}

	icalEvents, err := icalTodayEvents()
	if err == nil {
		events = append(events, icalEvents...)
	}

	// Sort by start time.
	sortEvents(events)
	return events, nil
}

func sortEvents(events []*Event) {
	for i := 1; i < len(events); i++ {
		for j := i; j > 0 && events[j].Start.Before(events[j-1].Start); j-- {
			events[j], events[j-1] = events[j-1], events[j]
		}
	}
}
