package cmd

import (
	"fmt"
	"strings"

	"github.com/benfourie/fl/internal/git"
	"github.com/benfourie/fl/internal/jira"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note <text>",
	Short: "Add a comment to the current Jira ticket",
	Long: `Adds a comment to the Jira ticket inferred from the current git branch.

Examples:
  fl note "Fixed the null pointer, needs QA review"
  fl note "Blocked by PROJ-456"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNote,
}

func runNote(cmd *cobra.Command, args []string) error {
	key, err := git.TicketKeyFromBranch()
	if err != nil {
		return fmt.Errorf("could not infer ticket key from branch: %w", err)
	}

	text := strings.Join(args, " ")

	client, err := jira.NewClient()
	if err != nil {
		return fmt.Errorf("jira: %w", err)
	}

	if err := client.AddComment(key, text); err != nil {
		return fmt.Errorf("adding comment to %s: %w", key, err)
	}

	fmt.Printf("Note added to %s\n", key)
	return nil
}
