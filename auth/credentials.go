package auth

import (
	_ "embed"
	"encoding/json"
)

//go:embed credentials.json
var credentialsJSON []byte

// Credentials holds the OAuth2 client credentials from credentials.json
type Credentials struct {
	Installed struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
}

var creds Credentials

func init() {
	if err := json.Unmarshal(credentialsJSON, &creds); err != nil {
		panic("failed to parse embedded credentials.json: " + err.Error())
	}
}

// ClientID returns the OAuth2 client ID
func ClientID() string {
	return creds.Installed.ClientID
}

// ClientSecret returns the OAuth2 client secret
func ClientSecret() string {
	return creds.Installed.ClientSecret
}
