package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/benfo/fl/internal/git"
	"github.com/spf13/cobra"
)

var subtaskAssignMe bool

var subtaskCmd = &cobra.Command{
	Use:   "subtask [key] <summary>",
	Short: "Add a subtask to a tracker item",
	Long: `Adds a subtask to a tracker item.

For Jira, creates a child issue using the project's subtask type.
For Trello, adds a checklist item to the card (using an existing
checklist, or creating one named "Tasks" if none exists).

The item key is inferred from the current git branch when omitted.

Examples:
  fl subtask "write unit tests"
  fl subtask "write unit tests" --assign
  fl subtask PROJ-123 "write unit tests"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSubtask,
}

func init() {
	subtaskCmd.Flags().BoolVarP(&subtaskAssignMe, "assign", "a", false, "Assign the new subtask to yourself (Jira only)")
}

func runSubtask(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	var key, summary string
	if len(args) == 2 {
		key = args[0]
		summary = strings.TrimSpace(args[1])
	} else {
		// Single arg is the summary; infer key from branch.
		key, err = git.TicketKeyFromBranch(client.KeyPattern())
		if err != nil {
			return fmt.Errorf("no key provided and could not infer from branch: %w", err)
		}
		summary = strings.TrimSpace(args[0])
	}

	if summary == "" {
		return fmt.Errorf("summary cannot be empty")
	}

	item, err := client.AddSubtask(key, summary)
	if err != nil {
		return err
	}

	if item.Key != key {
		// Jira: a new child issue was created with its own key.
		fmt.Printf("Created subtask %s on %s: %s\n", item.Key, key, item.Summary)
		if url, err := client.ItemURL(item.Key); err == nil {
			fmt.Println(" ", url)
		}
		if subtaskAssignMe {
			if err := client.AssignToMe(item.Key); err != nil {
				fmt.Fprintf(os.Stderr, "warning: could not assign subtask: %v\n", err)
			} else {
				fmt.Println("Assigned to you.")
			}
		}
	} else {
		// Trello: added as a checklist item on the same card.
		fmt.Printf("Added checklist item to %s: %s\n", key, item.Summary)
		if url, err := client.ItemURL(key); err == nil {
			fmt.Println(" ", url)
		}
	}
	return nil
}
