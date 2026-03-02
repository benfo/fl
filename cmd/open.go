package cmd

import (
	"fmt"

	"github.com/benfo/flow-cli/internal/git"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [item-key]",
	Short: "Open the current tracker item in your browser",
	Long: `Infers the item key from the current git branch and opens it
in your default browser.

Examples:
  fl open            # infers key from current branch
  fl open PROJ-123
  fl open abc12345   # Trello shortLink`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	var key string
	if len(args) == 1 {
		key = args[0]
	} else {
		key, err = git.TicketKeyFromBranch(client.KeyPattern())
		if err != nil {
			return fmt.Errorf("could not infer key from branch: %w", err)
		}
	}

	url, err := client.ItemURL(key)
	if err != nil {
		return err
	}

	if err := openBrowser(url); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}

	fmt.Printf("Opening %s\n", url)
	return nil
}
