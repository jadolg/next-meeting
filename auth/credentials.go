package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const appName = "next-meeting"
const credentialsFileName = "credentials.json"

// Credentials holds the OAuth2 client credentials from credentials.json
type Credentials struct {
	Installed struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"installed"`
}

var creds *Credentials

// GetCredentialsPath returns the path where credentials.json should be stored
func GetCredentialsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, appName, credentialsFileName), nil
}

// loadCredentials loads credentials from the config directory
func loadCredentials() error {
	if creds != nil {
		return nil
	}

	credPath, err := GetCredentialsPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(credPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("credentials file not found at %s. Please download your OAuth credentials from Google Cloud Console and save them there", credPath)
		}
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	var c Credentials
	if err := json.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("failed to parse credentials.json: %w", err)
	}

	creds = &c
	return nil
}

// ClientID returns the OAuth2 client ID
func ClientID() string {
	if err := loadCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return creds.Installed.ClientID
}

// ClientSecret returns the OAuth2 client secret
func ClientSecret() string {
	if err := loadCredentials(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	return creds.Installed.ClientSecret
}
