package config

import (
	"fmt"

	"github.com/spf13/viper"
	"github.com/zalando/go-keyring"
)

const (
	trelloAPIKeyKey = "trello-api-key"
	trelloTokenKey  = "trello-token"
)

func TrelloAPIKey() (string, error) {
	v, err := keyring.Get(keyringService, trelloAPIKeyKey)
	if err != nil {
		return "", fmt.Errorf("trello API key not found — run: fl auth trello")
	}
	return v, nil
}

func TrelloToken() (string, error) {
	v, err := keyring.Get(keyringService, trelloTokenKey)
	if err != nil {
		return "", fmt.Errorf("trello token not found — run: fl auth trello")
	}
	return v, nil
}

func SaveTrelloCredentials(apiKey, token string) error {
	if err := keyring.Set(keyringService, trelloAPIKeyKey, apiKey); err != nil {
		return fmt.Errorf("saving trello API key: %w", err)
	}
	if err := keyring.Set(keyringService, trelloTokenKey, token); err != nil {
		return fmt.Errorf("saving trello token: %w", err)
	}
	return nil
}

// TrelloBoardIDs returns the board IDs to restrict fl today to.
// Returns nil when unconfigured, meaning all boards are included.
func TrelloBoardIDs() []string {
	return viper.GetStringSlice("tracker.trello.board_ids")
}
