package config

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	outlookClientIDKey = "outlook-client-id"
	outlookTokenKey    = "outlook-token"
)

func OutlookClientID() (string, error) {
	v, err := keyring.Get(keyringService, outlookClientIDKey)
	if err != nil {
		return "", fmt.Errorf("outlook client ID not found — run: fl auth outlook")
	}
	return v, nil
}

func SaveOutlookClientID(clientID string) error {
	if err := keyring.Set(keyringService, outlookClientIDKey, clientID); err != nil {
		return fmt.Errorf("saving outlook client ID: %w", err)
	}
	return nil
}

func OutlookToken() (*oauth2.Token, error) {
	raw, err := keyring.Get(keyringService, outlookTokenKey)
	if err != nil {
		return nil, fmt.Errorf("outlook token not found — run: fl auth outlook")
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return nil, fmt.Errorf("corrupted outlook token — run: fl auth outlook")
	}
	return &tok, nil
}

func SaveOutlookToken(tok *oauth2.Token) error {
	raw, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("serializing outlook token: %w", err)
	}
	return keyring.Set(keyringService, outlookTokenKey, string(raw))
}
