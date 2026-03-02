package jira

import (
	"fmt"
	"strings"

	"github.com/benfourie/fl/internal/config"
	"github.com/go-resty/resty/v2"
)

// Client is a thin wrapper around the Jira Cloud REST API v3.
type Client struct {
	http  *resty.Client
	host  string
	email string
}

// Ticket represents the fields of a Jira issue we care about.
type Ticket struct {
	Key     string
	Summary string
	Status  string
	Type    string
}

// Transition is a Jira workflow transition.
type Transition struct {
	ID   string
	Name string
}

// NewClient constructs a Client using credentials from config/keychain.
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

// GetTicket fetches a single issue by key.
func (c *Client) GetTicket(key string) (*Ticket, error) {
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

	return &Ticket{
		Key:     result.Key,
		Summary: result.Fields.Summary,
		Status:  result.Fields.Status.Name,
		Type:    result.Fields.IssueType.Name,
	}, nil
}

// MyOpenTickets returns tickets assigned to the authenticated user that are
// in progress or to do, optionally filtered to configured projects.
func (c *Client) MyOpenTickets() ([]*Ticket, error) {
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

	tickets := make([]*Ticket, 0, len(result.Issues))
	for _, issue := range result.Issues {
		tickets = append(tickets, &Ticket{
			Key:     issue.Key,
			Summary: issue.Fields.Summary,
			Status:  issue.Fields.Status.Name,
			Type:    issue.Fields.IssueType.Name,
		})
	}
	return tickets, nil
}

// AddComment posts a plain-text comment to a ticket.
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

// GetTransitions returns the available workflow transitions for a ticket.
func (c *Client) GetTransitions(key string) ([]*Transition, error) {
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

	transitions := make([]*Transition, 0, len(result.Transitions))
	for _, t := range result.Transitions {
		transitions = append(transitions, &Transition{ID: t.ID, Name: t.Name})
	}
	return transitions, nil
}

// DoTransition executes a workflow transition on a ticket.
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

// buildMyTicketsJQL constructs the JQL for listing the current user's open
// tickets. When projects is non-empty a project filter is injected.
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

// TicketURL returns the browser URL for a given ticket key.
func TicketURL(key string) (string, error) {
	host := config.JiraHost()
	if host == "" {
		return "", fmt.Errorf("jira host not configured — run: fl auth jira")
	}
	return fmt.Sprintf("%s/browse/%s", strings.TrimRight(host, "/"), key), nil
}
