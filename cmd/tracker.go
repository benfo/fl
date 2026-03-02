package cmd

import (
	"fmt"

	"github.com/benfourie/fl/internal/config"
	"github.com/benfourie/fl/internal/jira"
	"github.com/benfourie/fl/internal/tracker"
	"github.com/benfourie/fl/internal/trello"
)

// newTrackerClient returns the configured tracker backend.
func newTrackerClient() (tracker.Client, error) {
	switch config.TrackerProvider() {
	case "trello":
		return trello.NewClient()
	case "jira", "":
		return jira.NewClient()
	default:
		return nil, fmt.Errorf("unknown tracker provider %q — supported: jira, trello", config.TrackerProvider())
	}
}
