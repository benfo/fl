package config

import (
	"encoding/json"
	"fmt"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	googleClientIDKey     = "google-client-id"
	googleClientSecretKey = "google-client-secret"
	googleTokenKey        = "google-token"
)

func GoogleClientID() (string, error) {
	v, err := keyring.Get(keyringService, googleClientIDKey)
	if err != nil {
		return "", fmt.Errorf("google client ID not found — run: fl auth google")
	}
	return v, nil
}

func GoogleClientSecret() (string, error) {
	v, err := keyring.Get(keyringService, googleClientSecretKey)
	if err != nil {
		return "", fmt.Errorf("google client secret not found — run: fl auth google")
	}
	return v, nil
}

func SaveGoogleCredentials(clientID, clientSecret string) error {
	if err := keyring.Set(keyringService, googleClientIDKey, clientID); err != nil {
		return fmt.Errorf("saving google client ID: %w", err)
	}
	if err := keyring.Set(keyringService, googleClientSecretKey, clientSecret); err != nil {
		return fmt.Errorf("saving google client secret: %w", err)
	}
	return nil
}

func GoogleToken() (*oauth2.Token, error) {
	raw, err := keyring.Get(keyringService, googleTokenKey)
	if err != nil {
		return nil, fmt.Errorf("google token not found — run: fl auth google")
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(raw), &tok); err != nil {
		return nil, fmt.Errorf("corrupted google token — run: fl auth google")
	}
	return &tok, nil
}

func SaveGoogleToken(tok *oauth2.Token) error {
	raw, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("serializing google token: %w", err)
	}
	return keyring.Set(keyringService, googleTokenKey, string(raw))
}
