package cmd

import (
	"fmt"

	"github.com/benfourie/fl/internal/browser"
	"github.com/benfourie/fl/internal/git"
)

// resolveTicketKey returns the ticket key from args or infers it from the
// current git branch when no args are provided.
func resolveTicketKey(args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	key, err := git.TicketKeyFromBranch()
	if err != nil {
		return "", fmt.Errorf("no ticket key provided and could not infer from branch: %w", err)
	}
	return key, nil
}

// openBrowser opens the given URL in the default system browser.
func openBrowser(rawURL string) error {
	return browser.Open(rawURL)
}
