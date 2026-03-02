package cmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/benfo/fl/internal/git"
	"github.com/spf13/cobra"
)

var prBase string
var prDraft bool

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Open a pull request for the current branch",
	Long: `Opens a new pull request (or merge request) in your browser,
pre-filled with the current branch's tracker item key and summary.

Supports GitHub and GitLab remotes (SSH and HTTPS).`,
	Args: cobra.NoArgs,
	RunE: runPR,
}

func init() {
	prCmd.Flags().StringVarP(&prBase, "base", "b", "", "Base branch to merge into (default: repo default branch)")
	prCmd.Flags().BoolVarP(&prDraft, "draft", "d", false, "Open as a draft pull request")
}

func runPR(cmd *cobra.Command, args []string) error {
	client, err := newTrackerClient()
	if err != nil {
		return err
	}

	branch, err := git.CurrentBranch()
	if err != nil {
		return err
	}
	if branch == "HEAD" {
		return fmt.Errorf("not on a branch (detached HEAD state)")
	}

	key, err := git.TicketKeyFromBranch(client.KeyPattern())
	if err != nil {
		return fmt.Errorf("could not infer tracker key from branch: %w", err)
	}

	item, err := client.GetItem(key)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", key, err)
	}

	remoteURL, err := git.RemoteURL("origin")
	if err != nil {
		return err
	}

	host, owner, repo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return err
	}

	base := prBase
	if base == "" {
		base = git.DefaultBaseBranch()
	}

	title := fmt.Sprintf("%s: %s", item.Key, item.Summary)
	if prDraft && strings.Contains(host, "gitlab") {
		title = "Draft: " + title
	}

	body, _ := client.ItemURL(item.Key)

	prURL := buildPRURL(host, owner, repo, base, branch, title, body, prDraft)

	fmt.Printf("Opening PR: %s → %s\n", branch, base)
	fmt.Printf("Title:      %s\n", title)
	return openBrowser(prURL)
}

// buildPRURL constructs a pre-filled PR/MR creation URL for GitHub or GitLab.
func buildPRURL(host, owner, repo, base, head, title, body string, draft bool) string {
	repoBase := fmt.Sprintf("https://%s/%s/%s", host, owner, repo)

	if strings.Contains(host, "gitlab") {
		q := url.Values{}
		q.Set("merge_request[source_branch]", head)
		q.Set("merge_request[target_branch]", base)
		q.Set("merge_request[title]", title)
		if body != "" {
			q.Set("merge_request[description]", body)
		}
		return repoBase + "/-/merge_requests/new?" + q.Encode()
	}

	// GitHub (github.com or GitHub Enterprise)
	q := url.Values{}
	q.Set("quick_pull", "1")
	q.Set("title", title)
	if body != "" {
		q.Set("body", body)
	}
	if draft {
		q.Set("draft", "1")
	}
	return fmt.Sprintf("%s/compare/%s...%s?%s", repoBase, base, head, q.Encode())
}
