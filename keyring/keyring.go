package keyring

import (
	"encoding/json"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	serviceName = "next-meeting"
	tokenKey    = "oauth-token"
)

// SaveToken stores the OAuth2 token in the system keyring
func SaveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keyring.Set(serviceName, tokenKey, string(data))
}

// LoadToken retrieves the OAuth2 token from the system keyring
func LoadToken() (*oauth2.Token, error) {
	data, err := keyring.Get(serviceName, tokenKey)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}

	return &token, nil
}

// DeleteToken removes the OAuth2 token from the system keyring
func DeleteToken() error {
	return keyring.Delete(serviceName, tokenKey)
}
