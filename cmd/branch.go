package cmd

import (
	"fmt"

	"github.com/benfo/fl/internal/git"
	"github.com/benfo/fl/internal/tracker"
	"github.com/benfo/fl/internal/ui"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch [item-key]",
	Short: "Create a git branch from a tracker item",
	Long: `Fetches the item and creates a local git branch.
The branch name is derived from your configured template.

When no key is provided and none can be inferred from the current branch,
your open items are listed so you can pick one interactively.

Examples:
  fl branch PROJ-123
  fl branch abc12345   # Trello shortLink
  fl branch            # infers key, or shows a picker`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBranch,
}

func runBranch(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	// Try to resolve the key from args or the current branch name.
	// If that fails, show a picker of open items instead.
	var item *tracker.Item

	key, keyErr := resolveTicketKey(args, client)
	if keyErr != nil {
		item, err = pickOpenItem(client)
		if err != nil {
			return err
		}
	} else {
		item, err = client.GetItem(key)
		if err != nil {
			return fmt.Errorf("fetching item %s: %w", key, err)
		}
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

// pickOpenItem fetches open items and shows an interactive picker.
func pickOpenItem(client tracker.Client) (*tracker.Item, error) {
	items, err := client.MyOpenItems()
	if err != nil {
		return nil, fmt.Errorf("fetching open items: %w", err)
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no open items found")
	}

	item, err := ui.PickItem(items)
	if err != nil {
		item, err = ui.PickItemFallback(items)
		if err != nil {
			return nil, err
		}
	}
	return item, nil
}
