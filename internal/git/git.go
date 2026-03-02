package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/benfourie/fl/internal/config"
	"github.com/benfourie/fl/internal/jira"
)

// ticketKeyPattern matches Jira-style keys like PROJ-123 anywhere in a string.
var ticketKeyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// TicketKeyFromBranch reads the current git branch and extracts a Jira ticket key.
func TicketKeyFromBranch() (string, error) {
	branch, err := currentBranch()
	if err != nil {
		return "", err
	}

	key := ticketKeyPattern.FindString(strings.ToUpper(branch))
	if key == "" {
		return "", fmt.Errorf("no Jira ticket key found in branch %q", branch)
	}
	return key, nil
}

// BranchName renders the configured branch name template for a given ticket.
func BranchName(ticket *jira.Ticket) (string, error) {
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
		Key:   ticket.Key,
		Title: slugify(ticket.Summary),
		Type:  issueTypeSlug(ticket.Type),
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

// currentBranch returns the name of the current git branch.
func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// slugify converts a string to a lowercase, hyphen-separated slug.
var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	// Truncate to keep branch names reasonable.
	if len(s) > 50 {
		s = s[:50]
		s = strings.TrimRight(s, "-")
	}
	return s
}

// issueTypeSlug maps Jira issue type names to short branch prefixes.
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
