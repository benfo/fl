package cmd

import (
	"fmt"

	"github.com/benfourie/fl/internal/git"
	"github.com/benfourie/fl/internal/jira"
	"github.com/spf13/cobra"
)

var openCmd = &cobra.Command{
	Use:   "open [ticket-key]",
	Short: "Open the current Jira ticket in your browser",
	Long: `Infers the Jira ticket key from the current git branch name
and opens it in your default browser.

Examples:
  fl open            # infers ticket from current branch
  fl open PROJ-123   # opens a specific ticket`,
	Args: cobra.MaximumNArgs(1),
	RunE: runOpen,
}

func runOpen(cmd *cobra.Command, args []string) error {
	var key string
	var err error

	if len(args) == 1 {
		key = args[0]
	} else {
		key, err = git.TicketKeyFromBranch()
		if err != nil {
			return fmt.Errorf("could not infer ticket key from branch: %w", err)
		}
	}

	url, err := jira.TicketURL(key)
	if err != nil {
		return err
	}

	if err := openBrowser(url); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}

	fmt.Printf("Opening %s\n", url)
	return nil
}
