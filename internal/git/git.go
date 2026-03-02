package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/benfourie/fl/internal/config"
	"github.com/benfourie/fl/internal/tracker"
)

// TicketKeyFromBranch reads the current git branch and extracts an item key
// using the regex supplied by the active tracker client.
// It tries the branch as-is first (handles Trello shortLinks), then uppercased
// (handles Jira-style keys like proj-123 written in lowercase).
func TicketKeyFromBranch(pattern *regexp.Regexp) (string, error) {
	branch, err := currentBranch()
	if err != nil {
		return "", err
	}

	if key := pattern.FindString(branch); key != "" {
		return key, nil
	}
	if key := pattern.FindString(strings.ToUpper(branch)); key != "" {
		return key, nil
	}

	return "", fmt.Errorf("no tracker key found in branch %q", branch)
}

// BranchName renders the configured branch name template for a given item.
func BranchName(item *tracker.Item) (string, error) {
	tmplStr := config.BranchTemplate()

	tmpl, err := template.New("branch").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("invalid branch template %q: %w", tmplStr, err)
	}

	data := struct {
		Key   string
		Title string
		Type  string
	}{
		Key:   item.Key,
		Title: slugify(item.Summary),
		Type:  issueTypeSlug(item.Type),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// CreateBranch creates and checks out a new local git branch.
func CreateBranch(name string) error {
	cmd := exec.Command("git", "checkout", "-b", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}
	return s
}

func issueTypeSlug(issueType string) string {
	switch strings.ToLower(issueType) {
	case "bug":
		return "fix"
	case "story", "feature":
		return "feat"
	case "task":
		return "chore"
	case "epic":
		return "epic"
	default:
		return "feat"
	}
}
