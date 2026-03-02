package calendar

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/benfourie/fl/internal/browser"
	"github.com/benfourie/fl/internal/config"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleCalendarScope = "https://www.googleapis.com/auth/calendar.readonly"

// SetupGoogleAuth runs the full interactive OAuth consent flow and persists
// the resulting credentials and tokens to the OS keychain.
func SetupGoogleAuth() error {
	fmt.Println("To connect Google Calendar, you need an OAuth 2.0 client ID.")
	fmt.Println()
	fmt.Println("Steps:")
	fmt.Println("  1. Go to https://console.cloud.google.com/apis/credentials")
	fmt.Println("  2. Create a project (if you don't have one)")
	fmt.Println("  3. Enable the Google Calendar API")
	fmt.Println("  4. Create credentials → OAuth client ID → Desktop app")
	fmt.Println("  5. Paste the client ID and secret below")
	fmt.Println()

	var clientID, clientSecret string

	fmt.Print("Client ID: ")
	fmt.Scanln(&clientID)

	fmt.Print("Client Secret: ")
	fmt.Scanln(&clientSecret)

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID and secret cannot be empty")
	}

	tok, err := runConsentFlow(clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("OAuth consent flow: %w", err)
	}

	if err := config.SaveGoogleCredentials(clientID, clientSecret); err != nil {
		return err
	}
	if err := config.SaveGoogleToken(tok); err != nil {
		return fmt.Errorf("saving token: %w", err)
	}

	fmt.Println("\nGoogle Calendar connected successfully.")
	return nil
}

// runConsentFlow opens the browser to the Google consent page, waits for the
// redirect, and exchanges the code for a token.
func runConsentFlow(clientID, clientSecret string) (*oauth2.Token, error) {
	state, err := randomState()
	if err != nil {
		return nil, err
	}

	redirectURI, codeCh, errCh, shutdown, err := startCallbackServer(state)
	if err != nil {
		return nil, err
	}
	defer shutdown()

	cfg := oauthConfig(clientID, clientSecret, redirectURI)
	authURL := cfg.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Printf("\nOpening your browser for Google authorization...\n")
	fmt.Printf("If the browser doesn't open, visit:\n  %s\n\n", authURL)
	_ = browser.Open(authURL)

	code, err := waitForCode(codeCh, errCh)
	if err != nil {
		return nil, err
	}

	tok, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		return nil, fmt.Errorf("exchanging authorization code: %w", err)
	}
	return tok, nil
}

// googleTodayEvents fetches today's calendar events from the Google Calendar API.
func googleTodayEvents() ([]*Event, error) {
	ctx := context.Background()
	client, err := authenticatedClient(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	params := url.Values{}
	params.Set("timeMin", startOfDay(now).Format(time.RFC3339))
	params.Set("timeMax", endOfDay(now).Format(time.RFC3339))
	params.Set("singleEvents", "true")
	params.Set("orderBy", "startTime")
	params.Set("maxResults", "25")

	apiURL := "https://www.googleapis.com/calendar/v3/calendars/primary/events?" + params.Encode()

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("google calendar API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return nil, fmt.Errorf("google calendar API %d: %s", resp.StatusCode, errBody.Error.Message)
	}

	var result struct {
		Items []struct {
			Summary  string `json:"summary"`
			Location string `json:"location"`
			Start    struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"start"`
			End struct {
				DateTime string `json:"dateTime"`
				Date     string `json:"date"`
			} `json:"end"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding google response: %w", err)
	}

	events := make([]*Event, 0, len(result.Items))
	for _, item := range result.Items {
		start, err := parseGoogleTime(item.Start.DateTime, item.Start.Date)
		if err != nil {
			continue
		}
		end, err := parseGoogleTime(item.End.DateTime, item.End.Date)
		if err != nil {
			continue
		}
		events = append(events, &Event{
			Title:    item.Summary,
			Start:    start,
			End:      end,
			Location: item.Location,
			Provider: "google",
		})
	}
	return events, nil
}

// authenticatedClient returns an http.Client whose token is automatically
// refreshed and persisted to the keychain on each refresh.
func authenticatedClient(ctx context.Context) (*http.Client, error) {
	clientID, err := config.GoogleClientID()
	if err != nil {
		return nil, err
	}
	clientSecret, err := config.GoogleClientSecret()
	if err != nil {
		return nil, err
	}
	tok, err := config.GoogleToken()
	if err != nil {
		return nil, err
	}

	cfg := oauthConfig(clientID, clientSecret, "")
	base := cfg.TokenSource(ctx, tok)
	ts := &persistingTokenSource{base: base, last: tok.AccessToken}
	return oauth2.NewClient(ctx, ts), nil
}

// persistingTokenSource wraps a TokenSource and saves refreshed tokens to
// the OS keychain so future invocations don't need to re-authorize.
type persistingTokenSource struct {
	base oauth2.TokenSource
	last string // last known access token
}

func (s *persistingTokenSource) Token() (*oauth2.Token, error) {
	tok, err := s.base.Token()
	if err != nil {
		return nil, err
	}
	if tok.AccessToken != s.last {
		_ = config.SaveGoogleToken(tok) // best-effort; ignore save errors
		s.last = tok.AccessToken
	}
	return tok, nil
}

func oauthConfig(clientID, clientSecret, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{googleCalendarScope},
		Endpoint:     google.Endpoint,
		RedirectURL:  redirectURI,
	}
}

func parseGoogleTime(dateTime, date string) (time.Time, error) {
	if dateTime != "" {
		return time.Parse(time.RFC3339, dateTime)
	}
	// All-day events have only a date.
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 0, t.Location())
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating state: %w", err)
	}
	return hex.EncodeToString(b), nil
}
