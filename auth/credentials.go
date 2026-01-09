package auth

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
)

// Credentials holds the OAuth2 client credentials from credentials.json
type Credentials struct {
	// ClientID returns the OAuth2 client ID
	ClientID string
	// ClientSecret returns the OAuth2 client secret
	ClientSecret string
}

// ensure the type implements the interfaces
var (
	_ json.Unmarshaler = &Credentials{}
	_ json.Marshaler   = Credentials{}
)

// UnmarshalJSON implements [json.Unmarshaler].
func (c *Credentials) UnmarshalJSON(content []byte) error {
	var creds struct {
		Installed struct {
			ClientID     string `json:"client_id"`
			ClientSecret string `json:"client_secret"`
		} `json:"installed"`
	}
	if err := json.Unmarshal(content, &creds); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}
	if creds.Installed.ClientID == "" {
		return fmt.Errorf("invalid credentials: missing 'installed.client_id'")
	}
	if creds.Installed.ClientSecret == "" {
		return fmt.Errorf("invalid credentials: missing 'installed.client_secret'")
	}
	*c = Credentials{
		ClientID:     creds.Installed.ClientID,
		ClientSecret: creds.Installed.ClientSecret,
	}
	return nil
}

// MarshalJSON implements [json.Marshaler].
func (c Credentials) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"installed": map[string]any{
			"client_id":     c.ClientID,
			"client_secret": c.ClientSecret,
		},
	})
}

// LoadCredentialsFromFile read target file and returns the parsed app credentials.
func LoadCredentialsFromFile(path string) (Credentials, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, err
	}
	if len(bytes.TrimSpace(b)) == 0 {
		return Credentials{}, fmt.Errorf("file %q is empty", path)
	}
	var creds Credentials
	if err := json.Unmarshal(b, &creds); err != nil {
		return Credentials{}, fmt.Errorf("parse credentials: %w", err)
	}
	return creds, nil
}
