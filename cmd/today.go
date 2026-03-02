package cmd

import (
	"github.com/benfo/fl/internal/calendar"
	"github.com/benfo/fl/internal/ui"
	"github.com/spf13/cobra"
)

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's tasks and calendar events",
	Long: `Displays a combined view of:
  - Open items assigned to you (Jira or Trello)
  - Today's calendar events (Google Calendar, Outlook, and iCal feeds)`,
	Args: cobra.NoArgs,
	RunE: runToday,
}

func init() {
	todayCmd.Flags().BoolVarP(&calendar.ForceRefresh, "refresh", "r", false, "Bypass iCal cache and re-fetch all feeds")
}

func runToday(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	items, err := client.MyOpenItems()
	if err != nil {
		return err
	}

	events, err := calendar.TodayEvents()
	if err != nil {
		events = nil
	}

	return ui.RenderToday(items, events)
}
