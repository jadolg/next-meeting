package auth

import (
	"encoding/json"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

const (
	keyringServiceName    = "next-meeting"
	keyringTokenKey       = "oauth-token"
	keyringCredentialsKey = "app-credentials"
)

// SaveToken stores the OAuth2 token in the system keyring
func SaveToken(token *oauth2.Token) error {
	return keyringSaveJSON(keyringTokenKey, token)
}

// LoadToken retrieves the OAuth2 token from the system keyring
func LoadToken() (*oauth2.Token, error) {
	return keyringLoadJSON[*oauth2.Token](keyringTokenKey)
}

// ClearToken removes the OAuth2 token from the system keyring
func ClearToken() error {
	return keyring.Delete(keyringServiceName, keyringTokenKey)
}

// SaveCredentials stores the app credentials in the system keyring
func SaveCredentials(creds Credentials) error {
	return keyringSaveJSON(keyringCredentialsKey, creds)
}

// LoadCredentials retrieves the app credentials from the system keyring
func LoadCredentials() (Credentials, error) {
	return keyringLoadJSON[Credentials](keyringCredentialsKey)
}

// ClearCredentials removes the app credentials from the system keyring
func ClearCredentials() error {
	return keyring.Delete(keyringServiceName, keyringCredentialsKey)
}

func keyringSaveJSON(key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return keyring.Set(keyringServiceName, key, string(data))
}

func keyringLoadJSON[T any](key string) (T, error) {
	data, err := keyring.Get(keyringServiceName, key)
	if err != nil {
		var zero T
		return zero, err
	}

	var value T
	if err := json.Unmarshal([]byte(data), &value); err != nil {
		var zero T
		return zero, err
	}

	return value, nil
}
