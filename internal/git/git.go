package git

import (
	"bytes"
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"text/template"

	"github.com/benfo/fl/internal/config"
	"github.com/benfo/fl/internal/tracker"
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

// maxBranchLen is the maximum length for a generated branch name.
// Keeps names readable in terminals and CI logs, and well clear of
// filesystem path-length limits (the Windows 260-char MAX_PATH limit is
// easily hit once git adds its refs/remotes/origin/ prefix).
const maxBranchLen = 72

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

	name := buf.String()
	if len(name) > maxBranchLen {
		name = strings.TrimRight(name[:maxBranchLen], "-/")
	}
	return name, nil
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

// CheckoutBranch switches to an existing local git branch.
func CheckoutBranch(name string) error {
	cmd := exec.Command("git", "checkout", name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// LocalBranches returns the set of local branch names as a map (name→name).
func LocalBranches() (map[string]string, error) {
	out, err := exec.Command("git", "branch").Output()
	if err != nil {
		return map[string]string{}, nil
	}
	branches := make(map[string]string)
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(strings.TrimPrefix(line, "*"))
		if name != "" {
			branches[name] = name
		}
	}
	return branches, nil
}

// IsWorkingTreeDirty reports whether there are uncommitted changes.
func IsWorkingTreeDirty() (bool, error) {
	out, err := exec.Command("git", "status", "--porcelain").Output()
	if err != nil {
		return false, fmt.Errorf("git status failed: %w", err)
	}
	return strings.TrimSpace(string(out)) != "", nil
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the name of the currently checked-out branch.
func CurrentBranch() (string, error) {
	return currentBranch()
}

// RemoteURL returns the fetch URL configured for the given remote (e.g. "origin").
func RemoteURL(remote string) (string, error) {
	out, err := exec.Command("git", "remote", "get-url", remote).Output()
	if err != nil {
		return "", fmt.Errorf("remote %q not found — is this a git repository with an origin?", remote)
	}
	return strings.TrimSpace(string(out)), nil
}

// ParseRemoteURL extracts the host, owner, and repo from a git remote URL.
// Handles SSH (git@github.com:owner/repo.git) and HTTPS formats.
func ParseRemoteURL(rawURL string) (host, owner, repo string, err error) {
	rawURL = strings.TrimSuffix(rawURL, ".git")

	// SSH: git@github.com:owner/repo
	if strings.HasPrefix(rawURL, "git@") {
		rawURL = strings.TrimPrefix(rawURL, "git@")
		parts := strings.SplitN(rawURL, ":", 2)
		if len(parts) != 2 {
			return "", "", "", fmt.Errorf("unrecognised SSH remote: %s", rawURL)
		}
		pathParts := strings.SplitN(parts[1], "/", 2)
		if len(pathParts) != 2 {
			return "", "", "", fmt.Errorf("unrecognised SSH remote path: %s", parts[1])
		}
		return parts[0], pathParts[0], pathParts[1], nil
	}

	// HTTPS: https://github.com/owner/repo
	parsed, parseErr := url.Parse(rawURL)
	if parseErr != nil {
		return "", "", "", fmt.Errorf("unrecognised remote URL: %s", rawURL)
	}
	pathParts := strings.SplitN(strings.TrimPrefix(parsed.Path, "/"), "/", 2)
	if len(pathParts) != 2 {
		return "", "", "", fmt.Errorf("unrecognised remote path: %s", parsed.Path)
	}
	return parsed.Host, pathParts[0], pathParts[1], nil
}

// DefaultBaseBranch returns the repo's default branch by inspecting the local
// remote-tracking ref. Falls back to "main" if unavailable.
func DefaultBaseBranch() string {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "--short").Output()
	if err != nil {
		return "main"
	}
	// "origin/main" → "main"
	parts := strings.SplitN(strings.TrimSpace(string(out)), "/", 2)
	if len(parts) == 2 {
		return parts[1]
	}
	return "main"
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
