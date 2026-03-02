package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/benfourie/fl/internal/ui"
	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create [summary]",
	Short: "Create a new item in the current tracker",
	Long: `Creates a new work item (Jira issue or Trello card) in the current tracker.

Pass the summary as an argument or leave it out to be prompted.
When multiple destinations are available (e.g. multiple projects or
board lists) an interactive picker is shown.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	// Resolve summary: from arg, or prompt.
	summary := ""
	if len(args) > 0 {
		summary = strings.TrimSpace(args[0])
	}
	if summary == "" {
		fmt.Print("Summary: ")
		r := bufio.NewReader(os.Stdin)
		line, _ := r.ReadString('\n')
		summary = strings.TrimSpace(line)
	}
	if summary == "" {
		return fmt.Errorf("summary cannot be empty")
	}

	dests, err := client.CreateDests()
	if err != nil {
		return err
	}
	if len(dests) == 0 {
		return fmt.Errorf("no create destinations found")
	}

	// Choose destination: skip picker when there's only one option.
	dest := dests[0]
	if len(dests) > 1 {
		dest, err = ui.PickCreateDest(dests)
		if err != nil {
			// Fall back to plain-text if TUI unavailable.
			dest, err = ui.PickCreateDestFallback(dests)
			if err != nil {
				return err
			}
		}
	}

	item, err := client.CreateItem(dest.ID, summary)
	if err != nil {
		return err
	}

	fmt.Printf("Created %s: %s\n", item.Key, item.Summary)
	if url, err := client.ItemURL(item.Key); err == nil {
		fmt.Println(url)
	}
	return nil
}
