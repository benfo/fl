package cmd

import (
	"fmt"

	"github.com/benfo/fl/internal/config"
	"github.com/benfo/fl/internal/jira"
	"github.com/benfo/fl/internal/tracker"
	"github.com/benfo/fl/internal/trello"
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
