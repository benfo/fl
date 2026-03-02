package cmd

import (
	"fmt"

	"github.com/benfo/fl/internal/config"
	"github.com/spf13/cobra"
)

var icalCmd = &cobra.Command{
	Use:   "ical",
	Short: "Manage iCal feed subscriptions",
	Long: `Subscribe to iCal feeds from any calendar provider.
Feeds are stored in ~/.fl/config.yaml and read on every fl today invocation.`,
}

var icalAddCmd = &cobra.Command{
	Use:   "add <name> <url>",
	Short: "Add an iCal feed",
	Long: `Subscribe to an iCal feed URL.

The URL can use https:// or webcal:// schemes.
Most calendar apps expose an iCal URL under their sharing or export settings.

Examples:
  fl ical add "Work"     https://calendar.google.com/calendar/ical/.../basic.ics
  fl ical add "Personal" webcal://p06-caldav.icloud.com/...`,
	Args: cobra.ExactArgs(2),
	RunE: runICalAdd,
}

var icalListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured iCal feeds",
	Args:  cobra.NoArgs,
	RunE:  runICalList,
}

var icalRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Remove an iCal feed by name",
	Args:    cobra.ExactArgs(1),
	RunE:    runICalRemove,
}

func init() {
	icalCmd.AddCommand(icalAddCmd)
	icalCmd.AddCommand(icalListCmd)
	icalCmd.AddCommand(icalRemoveCmd)
}

func runICalAdd(cmd *cobra.Command, args []string) error {
	name, url := args[0], args[1]
	if err := config.AddICalFeed(name, url); err != nil {
		return err
	}
	fmt.Printf("Added iCal feed %q\n", name)
	return nil
}

func runICalList(cmd *cobra.Command, args []string) error {
	feeds, err := config.ICalFeeds()
	if err != nil {
		return err
	}
	if len(feeds) == 0 {
		fmt.Println("No iCal feeds configured. Add one with: fl ical add <name> <url>")
		return nil
	}
	for _, f := range feeds {
		fmt.Printf("  %-20s %s\n", f.Name, f.URL)
	}
	return nil
}

func runICalRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	if err := config.RemoveICalFeed(name); err != nil {
		return err
	}
	fmt.Printf("Removed iCal feed %q\n", name)
	return nil
}
