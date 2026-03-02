package cmd

import (
	"fmt"

	"github.com/benfo/flow-cli/internal/browser"
	"github.com/benfo/flow-cli/internal/git"
	"github.com/benfo/flow-cli/internal/tracker"
)

// resolveTicketKey returns the item key from args, or infers it from the
// current git branch using the tracker client's key pattern.
func resolveTicketKey(args []string, client tracker.Client) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	key, err := git.TicketKeyFromBranch(client.KeyPattern())
	if err != nil {
		return "", fmt.Errorf("no key provided and could not infer from branch: %w", err)
	}
	return key, nil
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(rawURL string) error {
	return browser.Open(rawURL)
}
