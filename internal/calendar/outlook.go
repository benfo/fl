package calendar

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/benfo/flow-cli/internal/browser"
	"github.com/benfo/flow-cli/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/microsoft"
)

var outlookScopes = []string{
	"Calendars.Read",
	"offline_access",
	"User.Read",
}

// SetupOutlookAuth runs the interactive OAuth consent flow for Outlook/MS 365
// and persists the client ID and token to the OS keychain.
func SetupOutlookAuth() error {
	fmt.Println("To connect Outlook/MS 365 Calendar, you need an Azure app registration.")
	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("  1. Go to https://portal.azure.com/#view/Microsoft_AAD_RegisteredApps")
	fmt.Println("  2. New registration → name it anything (e.g. \"fl\")")
	fmt.Println("  3. Supported account types: \"Accounts in any organizational directory")
	fmt.Println("     and personal Microsoft accounts\"")
	fmt.Println("  4. Redirect URI: choose \"Mobile and desktop applications\",")
	fmt.Println("     enter  http://localhost")
	fmt.Println("  5. Register, then copy the Application (client) ID")
	fmt.Println()

	var clientID string
	fmt.Print("Application (client) ID: ")
	fmt.Scanln(&clientID)
	clientID = strings.TrimSpace(clientID)

	if clientID == "" {
		return fmt.Errorf("client ID cannot be empty")
	}

	tok, err := outlookConsentFlow(clientID)
	if err != nil {
		return fmt.Errorf("OAuth consent flow: %w", err)
	}

	if err := config.SaveOutlookClientID(clientID); err != nil {
		return err
	}
	if err := config.SaveOutlookToken(tok); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	fmt.Println("\nOutlook Calendar connected successfully.")
	return nil
}

// outlookConsentFlow performs a PKCE authorization code flow. Microsoft public
// client apps don't use a client secret — PKCE is the recommended alternative.
func outlookConsentFlow(clientID string) (*oauth2.Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	redirectURI, codeCh, errCh, shutdown, err := startCallbackServer(state)
	if err != nil {
		return nil, err
	}
	defer shutdown()

	verifier := oauth2.GenerateVerifier()
	cfg := outlookOAuthConfig(clientID, redirectURI)
	authURL := cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))

	fmt.Printf("\nOpening your browser for Microsoft authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authURL)
	_ = browser.Open(authURL)

	code, err := waitForCode(codeCh, errCh)
	if err != nil {
		return nil, err
	}

	tok, err := cfg.Exchange(context.Background(), code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}
	return tok, nil
}

// outlookTodayEvents fetches today's events from Microsoft Graph calendarView.
func outlookTodayEvents() ([]*Event, error) {
	ctx := context.Background()
	client, err := outlookAuthenticatedClient(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	params := url.Values{}
	params.Set("startDateTime", startOfDay(now).UTC().Format(time.RFC3339))
	params.Set("endDateTime", endOfDay(now).UTC().Format(time.RFC3339))
	params.Set("$select", "subject,start,end,location")
	params.Set("$orderby", "start/dateTime")
	params.Set("$top", "25")

	apiURL := "https://graph.microsoft.com/v1.0/me/calendarView?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	// Ask Graph to return times in UTC so parsing is unambiguous.
	req.Header.Set("Prefer", `outlook.timezone="UTC"`)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("microsoft graph API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("microsoft graph API %d: %s", resp.StatusCode, errBody.Error.Message)
	}

	var result struct {
		Value []struct {
			Subject  string `json:"subject"`
			Location struct {
				DisplayName string `json:"displayName"`
			} `json:"location"`
			Start struct {
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				TimeZone string `json:"timeZone"`
			} `json:"end"`
		} `json:"value"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding graph response: %w", err)
	}

	events := make([]*Event, 0, len(result.Value))
	for _, item := range result.Value {
		start, err := parseMsDateTime(item.Start.DateTime, item.Start.TimeZone)
		if err != nil {
			continue
		}
		end, err := parseMsDateTime(item.End.DateTime, item.End.TimeZone)
		if err != nil {
			continue
		}
		events = append(events, &Event{
			Title:    item.Subject,
			Start:    start,
			End:      end,
			Location: item.Location.DisplayName,
			Provider: "outlook",
		})
	}
	return events, nil
}

// outlookAuthenticatedClient returns an http.Client that auto-refreshes the
// token and persists any refreshed token back to the keychain.
func outlookAuthenticatedClient(ctx context.Context) (*http.Client, error) {
	clientID, err := config.OutlookClientID()
	if err != nil {
		return nil, err
	}
	tok, err := config.OutlookToken()
	if err != nil {
		return nil, err
	}

	cfg := outlookOAuthConfig(clientID, "")
	base := cfg.TokenSource(ctx, tok)
	ts := &outlookTokenSource{base: base, last: tok.AccessToken}
	return oauth2.NewClient(ctx, ts), nil
}

// outlookTokenSource persists refreshed tokens to the keychain.
type outlookTokenSource struct {
	base oauth2.TokenSource
	last string
}

func (s *outlookTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		_ = config.SaveOutlookToken(tok)
		s.last = tok.AccessToken
	}
	return tok, nil
}

func outlookOAuthConfig(clientID, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		Scopes:      outlookScopes,
		Endpoint:    microsoft.AzureADEndpoint("common"),
		RedirectURL: redirectURI,
		// No ClientSecret: public client apps use PKCE instead.
	}
}

// parseMsDateTime parses the dateTime strings returned by Microsoft Graph.
// Graph returns "2024-01-01T09:00:00.0000000" (no tz suffix) when the
// Prefer: outlook.timezone header is set, so we attach the timezone manually.
func parseMsDateTime(s, tz string) (time.Time, error) {
	// Try standard RFC3339 first (covers cases where Graph includes offset).
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}

	// Graph omits the timezone suffix when Prefer header is honoured.
	formats := []string{
		"2006-01-02T15:04:05.9999999",
		"2006-01-02T15:04:05",
	}
	var t time.Time
	var parseErr error
	for _, f := range formats {
		t, parseErr = time.Parse(f, s)
		if parseErr == nil {
			break
		}
	}
	if parseErr != nil {
		return time.Time{}, fmt.Errorf("parsing %q: %w", s, parseErr)
	}

	if tz != "" {
		loc, err := time.LoadLocation(tz)
		if err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc), nil
		}
	}
	return t.UTC(), nil
}
