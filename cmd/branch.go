package cmd

import (
	"fmt"

	"github.com/benfourie/fl/internal/git"
	"github.com/benfourie/fl/internal/jira"
	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:   "branch [ticket-key]",
	Short: "Create a git branch from a Jira ticket",
	Long: `Fetches the Jira ticket and creates a local git branch.
The branch name is derived from your configured template.

Examples:
  fl branch PROJ-123
  fl branch          # prompts for ticket key`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBranch,
}

func runBranch(cmd *cobra.Command, args []string) error {
	key, err := resolveTicketKey(args)
	if err != nil {
		return err
	}

	client, err := jira.NewClient()
	if err != nil {
		return fmt.Errorf("jira: %w", err)
	}

	ticket, err := client.GetTicket(key)
	if err != nil {
		return fmt.Errorf("fetching ticket %s: %w", key, err)
	}

	branchName, err := git.BranchName(ticket)
	if err != nil {
		return fmt.Errorf("building branch name: %w", err)
	}

	if err := git.CreateBranch(branchName); err != nil {
		return fmt.Errorf("creating branch: %w", err)
	}

	fmt.Printf("Switched to new branch: %s\n", branchName)
	return nil
}
