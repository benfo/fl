package cmd

import (
	"fmt"

	"github.com/benfourie/fl/internal/jira"
	"github.com/benfourie/fl/internal/ui"
	"github.com/spf13/cobra"
)

var moveCmd = &cobra.Command{
	Use:   "move [ticket-key]",
	Short: "Move the current ticket to the next workflow step",
	Long: `Fetches available transitions for the Jira ticket and lets you
pick the next status. Infers ticket from the current git branch.

Examples:
  fl move             # infers ticket from branch, shows transition picker
  fl move PROJ-123    # move a specific ticket`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMove,
}

func runMove(cmd *cobra.Command, args []string) error {
	key, err := resolveTicketKey(args)
	if err != nil {
		return err
	}

	client, err := jira.NewClient()
	if err != nil {
		return fmt.Errorf("jira: %w", err)
	}

	transitions, err := client.GetTransitions(key)
	if err != nil {
		return fmt.Errorf("fetching transitions for %s: %w", key, err)
	}

	if len(transitions) == 0 {
		return fmt.Errorf("no available transitions for %s", key)
	}

	chosen, err := ui.PickTransition(transitions)
	if err != nil {
		return err
	}

	if err := client.DoTransition(key, chosen.ID); err != nil {
		return fmt.Errorf("moving %s: %w", key, err)
	}

	fmt.Printf("%s moved to: %s\n", key, chosen.Name)
	return nil
}
