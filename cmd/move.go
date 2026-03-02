package cmd

import (
	"fmt"

	"github.com/benfo/fl/internal/tracker"
	"github.com/benfo/fl/internal/ui"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move [item-key]",
	Short: "Move the current item to the next workflow step",
	Long: `Shows available transitions (Jira) or lists (Trello) and lets you
pick where to move the item. Infers the key from the current git branch.

Examples:
  fl move             # infers key from branch, shows picker
  fl move PROJ-123
  fl move abc12345    # Trello shortLink`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMove,
}

func runMove(cmd *cobra.Command, args []string) error {
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

	return runPickerTransition(key, transitions, client)
}

// runPickerTransition shows the interactive transition picker and applies the
// chosen transition. Shared by fl move and the fl done/fl start fallback.
func runPickerTransition(key string, transitions []*tracker.Transition, client tracker.Client) error {
	chosen, err := ui.PickTransition(transitions)
	if err != nil {
		chosen, err = ui.PickTransitionFallback(transitions)
		if err != nil {
			return err
		}
	}

	if err := client.DoTransition(key, chosen.ID); err != nil {
		return fmt.Errorf("moving %s: %w", key, err)
	}

	fmt.Printf("%s → %s\n", key, chosen.Name)
	return nil
}
