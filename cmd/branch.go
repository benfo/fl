package cmd

import (
	"fmt"

	"github.com/benfo/fl/internal/git"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch [item-key]",
	Short: "Create a git branch from a tracker item",
	Long: `Fetches the item and creates a local git branch.
The branch name is derived from your configured template.

Examples:
  fl branch PROJ-123
  fl branch abc12345   # Trello shortLink
  fl branch            # infers key from current branch`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBranch,
}

func runBranch(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	key, err := resolveTicketKey(args, client)
	if err != nil {
		return err
	}

	item, err := client.GetItem(key)
	if err != nil {
		return fmt.Errorf("fetching item %s: %w", key, err)
	}

	branchName, err := git.BranchName(item)
	if err != nil {
		return fmt.Errorf("building branch name: %w", err)
	}

	if err := git.CreateBranch(branchName); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	fmt.Printf("Switched to new branch: %s\n", branchName)
	return nil
}
