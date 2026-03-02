package cmd

import (
	"fmt"
	"strings"

	"github.com/benfo/fl/internal/tracker"
	"github.com/spf13/cobra"
)

var doneCmd = &cobra.Command{
	Use:   "done [item-key]",
	Short: "Mark the current item as done",
	Long: `Moves the item to its "done" state without showing the full picker.

Matches the first available transition whose name contains any of:
done, complete, closed, resolved, finish

If no match is found, falls back to showing all transitions so you
can pick manually (equivalent to fl move).

The item key is inferred from the current git branch when omitted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickTransition(args, []string{"done", "complete", "closed", "resolved", "finish"})
	},
}

var startCmd = &cobra.Command{
	Use:   "start [item-key]",
	Short: "Mark the current item as in progress",
	Long: `Moves the item to its "in progress" state without showing the full picker.

Matches the first available transition whose name contains any of:
progress, doing, started, active, start, development, dev

If no match is found, falls back to showing all transitions so you
can pick manually (equivalent to fl move).

The item key is inferred from the current git branch when omitted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runQuickTransition(args, []string{"progress", "doing", "started", "active", "start", "development", "dev"})
	},
}

// runQuickTransition applies the first transition whose name contains one of
// the given keywords (case-insensitive). Falls back to the interactive picker
// when no keyword matches.
func runQuickTransition(args []string, keywords []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	key, err := resolveTicketKey(args, client)
	if err != nil {
		return err
	}

	transitions, err := client.GetTransitions(key)
	if err != nil {
		return fmt.Errorf("fetching transitions for %s: %w", key, err)
	}
	if len(transitions) == 0 {
		return fmt.Errorf("no available transitions for %s", key)
	}

	match := findTransition(transitions, keywords)
	if match != nil {
		if err := client.DoTransition(key, match.ID); err != nil {
			return fmt.Errorf("moving %s: %w", key, err)
		}
		fmt.Printf("%s → %s\n", key, match.Name)
		return nil
	}

	// No keyword match — fall back to the interactive picker with a hint.
	fmt.Printf("No matching transition found. Available transitions for %s:\n", key)
	return runPickerTransition(key, transitions, client)
}

// findTransition returns the first transition whose lowercased name contains
// any of the given keywords, or nil if none match.
func findTransition(transitions []*tracker.Transition, keywords []string) *tracker.Transition {
	for _, t := range transitions {
		lower := strings.ToLower(t.Name)
		for _, kw := range keywords {
			if strings.Contains(lower, kw) {
				return t
			}
		}
	}
	return nil
}
