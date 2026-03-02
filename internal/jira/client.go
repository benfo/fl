package jira

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/benfo/flow-cli/internal/config"
	"github.com/benfo/flow-cli/internal/tracker"
	"github.com/go-resty/resty/v2"
)

var keyPattern = regexp.MustCompile(`[A-Z][A-Z0-9]+-\d+`)

// Client is a thin wrapper around the Jira Cloud REST API v3.
// It implements tracker.Client.
type Client struct {
	http  *resty.Client
	host  string
	email string
}

func NewClient() (*Client, error) {
	host := config.JiraHost()
	if host == "" {
		return nil, fmt.Errorf("jira host not configured — run: fl auth jira")
	}

	email := config.JiraEmail()
	token, err := config.JiraToken()
	if err != nil {
		return nil, err
	}

	http := resty.New().
		SetBaseURL(strings.TrimRight(host, "/")).
		SetBasicAuth(email, token).
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json")

	return &Client{http: http, host: host, email: email}, nil
}

func (c *Client) KeyPattern() *regexp.Regexp {
	return keyPattern
}

func (c *Client) GetItem(key string) (*tracker.Item, error) {
	var result struct {
		Key    string `json:"key"`
		Fields struct {
			Summary string `json:"summary"`
			Status  struct {
				Name string `json:"name"`
			} `json:"status"`
			IssueType struct {
				Name string `json:"name"`
			} `json:"issuetype"`
		} `json:"fields"`
	}

	resp, err := c.http.R().
		SetResult(&result).
		Get(fmt.Sprintf("/rest/api/3/issue/%s", key))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}

	return &tracker.Item{
		Key:     result.Key,
		Summary: result.Fields.Summary,
		Status:  result.Fields.Status.Name,
		Type:    result.Fields.IssueType.Name,
	}, nil
}

func (c *Client) ItemURL(key string) (string, error) {
	return fmt.Sprintf("%s/browse/%s", strings.TrimRight(c.host, "/"), key), nil
}

func (c *Client) MyOpenItems() ([]*tracker.Item, error) {
	jql := buildMyTicketsJQL(config.JiraProjects())

	var result struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Summary string `json:"summary"`
				Status  struct {
					Name string `json:"name"`
				} `json:"status"`
				IssueType struct {
					Name string `json:"name"`
				} `json:"issuetype"`
			} `json:"fields"`
		} `json:"issues"`
	}

	resp, err := c.http.R().
		SetResult(&result).
		SetBody(map[string]any{
			"jql":        jql,
			"maxResults": 20,
			"fields":     []string{"summary", "status", "issuetype"},
		}).
		Post("/rest/api/3/search/jql")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}

	items := make([]*tracker.Item, 0, len(result.Issues))
	for _, issue := range result.Issues {
		items = append(items, &tracker.Item{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.IssueType.Name,
		})
	}
	return items, nil
}

func (c *Client) AddComment(key, text string) error {
	body := map[string]any{
		"body": map[string]any{
			"type":    "doc",
			"version": 1,
			"content": []map[string]any{
				{
					"type": "paragraph",
					"content": []map[string]any{
						{"type": "text", "text": text},
					},
				},
			},
		},
	}

	resp, err := c.http.R().
		SetBody(body).
		Post(fmt.Sprintf("/rest/api/3/issue/%s/comment", key))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) GetTransitions(key string) ([]*tracker.Transition, error) {
	var result struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"transitions"`
	}

	resp, err := c.http.R().
		SetResult(&result).
		Get(fmt.Sprintf("/rest/api/3/issue/%s/transitions", key))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}

	transitions := make([]*tracker.Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, &tracker.Transition{ID: t.ID, Name: t.Name})
	}
	return transitions, nil
}

func (c *Client) DoTransition(key, transitionID string) error {
	body := map[string]any{
		"transition": map[string]string{"id": transitionID},
	}

	resp, err := c.http.R().
		SetBody(body).
		Post(fmt.Sprintf("/rest/api/3/issue/%s/transitions", key))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) CreateDests() ([]*tracker.CreateDest, error) {
	projects := config.JiraProjects()
	if len(projects) == 0 {
		return nil, fmt.Errorf("no jira.projects configured — add project keys to ~/.fl/config.yaml or .fl.yaml")
	}

	var dests []*tracker.CreateDest
	for _, projectKey := range projects {
		types, err := c.projectIssueTypes(projectKey)
		if err != nil {
			return nil, fmt.Errorf("fetching issue types for %s: %w", projectKey, err)
		}
		for _, t := range types {
			if t.Subtask {
				continue
			}
			dests = append(dests, &tracker.CreateDest{
				ID:    projectKey + "\t" + t.Name,
				Label: projectKey + " · " + t.Name,
			})
		}
	}
	return dests, nil
}

func (c *Client) projectIssueTypes(projectKey string) ([]jiraIssueType, error) {
	var project struct {
		IssueTypes []jiraIssueType `json:"issueTypes"`
	}
	resp, err := c.http.R().
		SetResult(&project).
		Get(fmt.Sprintf("/rest/api/3/project/%s", projectKey))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}
	return project.IssueTypes, nil
}

type jiraIssueType struct {
	Name    string `json:"name"`
	Subtask bool   `json:"subtask"`
}

func (c *Client) CreateItem(destID, summary string) (*tracker.Item, error) {
	// destID is "<projectKey>\t<issueTypeName>"
	parts := strings.SplitN(destID, "\t", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid create destination: %s", destID)
	}
	projectKey, typeName := parts[0], parts[1]

	body := map[string]any{
		"fields": map[string]any{
			"project":   map[string]string{"key": projectKey},
			"issuetype": map[string]string{"name": typeName},
			"summary":   summary,
		},
	}

	var result struct {
		Key string `json:"key"`
	}
	resp, err := c.http.R().
		SetBody(body).
		SetResult(&result).
		Post("/rest/api/3/issue")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}

	return &tracker.Item{
		Key:     result.Key,
		Summary: summary,
		Type:    typeName,
		Status:  "To Do",
	}, nil
}

func (c *Client) AddSubtask(parentKey, summary string) (*tracker.Item, error) {
	// Derive project key from issue key, e.g. "PROJ-123" → "PROJ".
	projectKey := strings.SplitN(parentKey, "-", 2)[0]

	// Find the subtask issue type for this project.
	types, err := c.projectIssueTypes(projectKey)
	if err != nil {
		return nil, fmt.Errorf("fetching issue types for %s: %w", projectKey, err)
	}
	var subtaskTypeName string
	for _, t := range types {
		if t.Subtask {
			subtaskTypeName = t.Name
			break
		}
	}
	if subtaskTypeName == "" {
		return nil, fmt.Errorf("project %s does not have a subtask issue type — subtasks may not be enabled", projectKey)
	}

	body := map[string]any{
		"fields": map[string]any{
			"project":   map[string]string{"key": projectKey},
			"parent":    map[string]string{"key": parentKey},
			"issuetype": map[string]string{"name": subtaskTypeName},
			"summary":   summary,
		},
	}

	var result struct {
		Key string `json:"key"`
	}
	resp, err := c.http.R().
		SetBody(body).
		SetResult(&result).
		Post("/rest/api/3/issue")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("jira API %d: %s", resp.StatusCode(), resp.String())
	}

	return &tracker.Item{
		Key:     result.Key,
		Summary: summary,
		Type:    subtaskTypeName,
		Status:  "To Do",
	}, nil
}

func buildMyTicketsJQL(projects []string) string {
	base := `assignee = currentUser() AND statusCategory in ("In Progress", "To Do")`

	if len(projects) > 0 {
		quoted := make([]string, len(projects))
		for i, p := range projects {
			quoted[i] = `"` + p + `"`
		}
		base += ` AND project in (` + strings.Join(quoted, ", ") + `)`
	}

	return base + ` ORDER BY updated DESC`
}
