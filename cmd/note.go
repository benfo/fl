package cmd

import (
	"fmt"
	"strings"

	"github.com/benfo/fl/internal/git"
	"github.com/spf13/cobra"
)

var noteCmd = &cobra.Command{
	Use:   "note <text>",
	Short: "Add a comment to the current tracker item",
	Long: `Adds a comment to the item inferred from the current git branch.

Examples:
  fl note "Fixed the null pointer, needs QA review"
  fl note "Blocked by PROJ-456"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runNote,
}

func runNote(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	key, err := git.TicketKeyFromBranch(client.KeyPattern())
	if err != nil {
		return fmt.Errorf("could not infer key from branch: %w", err)
	}

	text := strings.Join(args, " ")

	if err := client.AddComment(key, text); err != nil {
		return fmt.Errorf("adding comment to %s: %w", key, err)
	}

	fmt.Printf("Note added to %s\n", key)
	return nil
}
