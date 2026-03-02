package cmd

import (
	"github.com/benfourie/fl/internal/calendar"
	"github.com/benfourie/fl/internal/jira"
	"github.com/benfourie/fl/internal/ui"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's Jira tasks and calendar events",
	Long: `Displays a combined view of:
  - Jira tickets assigned to you (in progress / to do)
  - Today's calendar events (Google Calendar, Outlook, and iCal feeds)`,
	Args: cobra.NoArgs,
	RunE: runToday,
}

func init() {
	todayCmd.Flags().BoolVarP(&calendar.ForceRefresh, "refresh", "r", false, "Bypass iCal cache and re-fetch all feeds")
}

func runToday(cmd *cobra.Command, args []string) error {
	jiraClient, err := jira.NewClient()
	if err != nil {
		return err
	}

	tickets, err := jiraClient.MyOpenTickets()
	if err != nil {
		return err
	}

	events, err := calendar.TodayEvents()
	if err != nil {
		// calendar errors are non-fatal; show what we have
		events = nil
	}

	return ui.RenderToday(tickets, events)
}
