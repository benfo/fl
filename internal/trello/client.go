package trello

import (
	"fmt"
	"regexp"

	"github.com/benfo/fl/internal/browser"
	"github.com/benfo/fl/internal/config"
	"github.com/benfo/fl/internal/tracker"
	"github.com/go-resty/resty/v2"
)

// Trello shortLinks are exactly 8 alphanumeric characters.
var keyPattern = regexp.MustCompile(`\b[a-zA-Z0-9]{8}\b`)

// Client implements tracker.Client against the Trello REST API v1.
type Client struct {
	http *resty.Client
}

func NewClient() (*Client, error) {
	apiKey, err := config.TrelloAPIKey()
	if err != nil {
		return nil, err
	}
	token, err := config.TrelloToken()
	if err != nil {
		return nil, err
	}

	http := resty.New().
		SetBaseURL("https://api.trello.com/1").
		SetQueryParam("key", apiKey).
		SetQueryParam("token", token).
		SetHeader("Accept", "application/json")

	return &Client{http: http}, nil
}

func (c *Client) KeyPattern() *regexp.Regexp {
	return keyPattern
}

// SetupAuth walks the user through obtaining a Trello API key and token.
func SetupAuth() error {
	fmt.Println("To connect Trello, you need an API key and a token.")
	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("  1. Get your API key from: https://trello.com/app-key")
	fmt.Println("  2. Paste it below — fl will open the authorization page for you")
	fmt.Println()

	var apiKey string
	fmt.Print("API key: ")
	fmt.Scanln(&apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	authURL := fmt.Sprintf(
		"https://trello.com/1/authorize?expiration=never&name=fl&scope=read%%2Cwrite&response_type=token&key=%s",
		apiKey,
	)
	fmt.Printf("\nOpening authorization page...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authURL)
	_ = browser.Open(authURL)

	fmt.Print("Paste the token Trello shows you: ")
	var token string
	fmt.Scanln(&token)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := config.SaveTrelloCredentials(apiKey, token); err != nil {
		return err
	}

	fmt.Println("\nTrello connected successfully.")
	return nil
}

func (c *Client) GetItem(key string) (*tracker.Item, error) {
	var card struct {
		ShortLink string `json:"shortLink"`
		Name      string `json:"name"`
		IdList    string `json:"idList"`
	}

	resp, err := c.http.R().
		SetResult(&card).
		SetQueryParam("fields", "name,shortLink,idList").
		Get(fmt.Sprintf("/cards/%s", key))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	listName, _ := c.listName(card.IdList)

	return &tracker.Item{
		Key:     card.ShortLink,
		Summary: card.Name,
		Status:  listName,
		Type:    "card",
	}, nil
}

func (c *Client) ItemURL(key string) (string, error) {
	return fmt.Sprintf("https://trello.com/c/%s", key), nil
}

func (c *Client) AddComment(key, text string) error {
	resp, err := c.http.R().
		SetQueryParam("text", text).
		Post(fmt.Sprintf("/cards/%s/actions/comments", key))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) GetTransitions(key string) ([]*tracker.Transition, error) {
	// Fetch the card to get its board, then return the board's lists as transitions.
	var card struct {
		IdBoard string `json:"idBoard"`
	}
	resp, err := c.http.R().
		SetResult(&card).
		SetQueryParam("fields", "idBoard").
		Get(fmt.Sprintf("/cards/%s", key))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	return c.boardLists(card.IdBoard)
}

func (c *Client) DoTransition(key, listID string) error {
	resp, err := c.http.R().
		SetQueryParam("idList", listID).
		Put(fmt.Sprintf("/cards/%s", key))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) MyOpenItems() ([]*tracker.Item, error) {
	var cards []struct {
		ShortLink string `json:"shortLink"`
		Name      string `json:"name"`
		IdList    string `json:"idList"`
		IdBoard   string `json:"idBoard"`
	}

	resp, err := c.http.R().
		SetResult(&cards).
		SetQueryParam("filter", "open").
		SetQueryParam("fields", "name,shortLink,idList,idBoard").
		Get("/members/me/cards")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	// Optionally restrict to configured board IDs.
	boardFilter := boardFilterSet(config.TrelloBoardIDs())

	// Collect the unique list IDs we need to resolve.
	listIDs := map[string]struct{}{}
	for _, card := range cards {
		if len(boardFilter) > 0 && !boardFilter[card.IdBoard] {
			continue
		}
		listIDs[card.IdList] = struct{}{}
	}

	// Batch-resolve list names (one request per unique list).
	listNames := make(map[string]string, len(listIDs))
	for id := range listIDs {
		name, err := c.listName(id)
		if err == nil {
			listNames[id] = name
		}
	}

	items := make([]*tracker.Item, 0, len(cards))
	for _, card := range cards {
		if len(boardFilter) > 0 && !boardFilter[card.IdBoard] {
			continue
		}
		items = append(items, &tracker.Item{
			Key:     card.ShortLink,
			Summary: card.Name,
			Status:  listNames[card.IdList],
			Type:    "card",
		})
	}
	return items, nil
}

// listName fetches the name of a Trello list by its ID.
func (c *Client) listName(listID string) (string, error) {
	var list struct {
		Name string `json:"name"`
	}
	resp, err := c.http.R().
		SetResult(&list).
		SetQueryParam("fields", "name").
		Get(fmt.Sprintf("/lists/%s", listID))
	if err != nil {
		return "", err
	}
	if resp.IsError() {
		return "", fmt.Errorf("trello API %d", resp.StatusCode())
	}
	return list.Name, nil
}

// boardLists returns all open lists on a board as Transitions.
func (c *Client) boardLists(boardID string) ([]*tracker.Transition, error) {
	var lists []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	resp, err := c.http.R().
		SetResult(&lists).
		SetQueryParam("filter", "open").
		Get(fmt.Sprintf("/boards/%s/lists", boardID))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	transitions := make([]*tracker.Transition, len(lists))
	for i, l := range lists {
		transitions[i] = &tracker.Transition{ID: l.ID, Name: l.Name}
	}
	return transitions, nil
}

func (c *Client) AssignToMe(cardKey string) error {
	// Get the current member's ID.
	var me struct {
		ID string `json:"id"`
	}
	resp, err := c.http.R().
		SetResult(&me).
		SetQueryParam("fields", "id").
		Get("/members/me")
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	resp, err = c.http.R().
		SetQueryParam("idMembers", me.ID).
		Put(fmt.Sprintf("/cards/%s", cardKey))
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}
	return nil
}

func (c *Client) AddSubtask(cardKey, summary string) (*tracker.Item, error) {
	// Get existing checklists on the card.
	var checklists []struct {
		ID string `json:"id"`
	}
	resp, err := c.http.R().
		SetResult(&checklists).
		Get(fmt.Sprintf("/cards/%s/checklists", cardKey))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	// Use the first existing checklist, or create one named "Tasks".
	var checklistID string
	if len(checklists) > 0 {
		checklistID = checklists[0].ID
	} else {
		var cl struct {
			ID string `json:"id"`
		}
		resp, err = c.http.R().
			SetResult(&cl).
			SetBody(map[string]string{"idCard": cardKey, "name": "Tasks"}).
			SetHeader("Content-Type", "application/json").
			Post("/checklists")
		if err != nil {
			return nil, err
		}
		if resp.IsError() {
			return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
		}
		checklistID = cl.ID
	}

	// Add the checklist item.
	resp, err = c.http.R().
		SetQueryParam("name", summary).
		Post(fmt.Sprintf("/checklists/%s/checkItems", checklistID))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}

	// Return the card key as item key — checklist items don't have independent URLs.
	return &tracker.Item{
		Key:     cardKey,
		Summary: summary,
		Type:    "checklist item",
	}, nil
}

func (c *Client) CreateDests() ([]*tracker.CreateDest, error) {
	boardIDs := config.TrelloBoardIDs()

	// If no boards are configured, discover the member's open boards.
	if len(boardIDs) == 0 {
		var boards []struct {
			ID string `json:"id"`
		}
		resp, err := c.http.R().
			SetResult(&boards).
			SetQueryParam("filter", "open").
			SetQueryParam("fields", "id").
			Get("/members/me/boards")
		if err != nil {
			return nil, err
		}
		if resp.IsError() {
			return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
		}
		for _, b := range boards {
			boardIDs = append(boardIDs, b.ID)
		}
	}

	var dests []*tracker.CreateDest
	for _, boardID := range boardIDs {
		var board struct {
			Name  string `json:"name"`
			Lists []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"lists"`
		}
		resp, err := c.http.R().
			SetResult(&board).
			SetQueryParam("lists", "open").
			SetQueryParam("fields", "name").
			Get(fmt.Sprintf("/boards/%s", boardID))
		if err != nil || resp.IsError() {
			continue
		}
		for _, l := range board.Lists {
			dests = append(dests, &tracker.CreateDest{
				ID:    l.ID,
				Label: board.Name + " · " + l.Name,
			})
		}
	}
	if len(dests) == 0 {
		return nil, fmt.Errorf("no open lists found — check your board configuration")
	}
	return dests, nil
}

func (c *Client) CreateItem(destID, summary string) (*tracker.Item, error) {
	// destID is the Trello list ID.
	var card struct {
		ShortLink string `json:"shortLink"`
	}
	resp, err := c.http.R().
		SetResult(&card).
		SetBody(map[string]string{
			"name":   summary,
			"idList": destID,
		}).
		SetHeader("Content-Type", "application/json").
		Post("/cards")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("trello API %d: %s", resp.StatusCode(), resp.String())
	}
	return &tracker.Item{
		Key:     card.ShortLink,
		Summary: summary,
		Type:    "card",
	}, nil
}

func boardFilterSet(ids []string) map[string]bool {
	if len(ids) == 0 {
		return nil
	}
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = true
	}
	return m
}
