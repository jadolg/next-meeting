package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"

	"next-meeting/keyring"
)

const redirectURL = "http://localhost:8085/callback"

func getOAuthConfig() (*oauth2.Config, error) {
	if err := loadCredentials(); err != nil {
		return nil, err
	}
	return &oauth2.Config{
		ClientID:     creds.Installed.ClientID,
		ClientSecret: creds.Installed.ClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{calendar.CalendarReadonlyScope},
		Endpoint:     google.Endpoint,
	}, nil
}

// loadValidToken loads the stored token, refreshes it if needed, and saves it back.
func loadValidToken(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	token, err := keyring.LoadToken()
	if err != nil || token == nil {
		return nil, fmt.Errorf("not logged in")
	}
	tokenSource := config.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("token expired, please login again")
	}
	if newToken.AccessToken != token.AccessToken {
		if saveErr := keyring.SaveToken(newToken); saveErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save refreshed token: %v\n", saveErr)
		}
	}
	return newToken, nil
}

// IsLoggedIn checks if we have a valid token stored
func IsLoggedIn(ctx context.Context) bool {
	config, err := getOAuthConfig()
	if err != nil {
		return false
	}
	_, err = loadValidToken(ctx, config)
	return err == nil
}

// GetClient returns an authenticated HTTP client.
// It first tries to load a token from the keyring.
// If no token exists or the token is invalid, it returns an error.
func GetClient(ctx context.Context) (*http.Client, error) {
	config, err := getOAuthConfig()
	if err != nil {
		return nil, err
	}
	token, err := loadValidToken(ctx, config)
	if err != nil {
		return nil, err
	}
	return config.Client(ctx, token), nil
}

// Login initiates the OAuth2 flow and saves the token
func Login(ctx context.Context) error {
	config, err := getOAuthConfig()
	if err != nil {
		return err
	}
	token, err := getTokenFromWeb(ctx, config)
	if err != nil {
		return fmt.Errorf("unable to get token from web: %w", err)
	}
	if err := keyring.SaveToken(token); err != nil {
		return fmt.Errorf("could not save token to keyring: %w", err)
	}
	return nil
}

// getTokenFromWeb starts a local HTTP server and initiates the OAuth2 flow
func getTokenFromWeb(ctx context.Context, config *oauth2.Config) (*oauth2.Token, error) {
	// Create a channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Create a simple HTTP server to handle the callback
	server := &http.Server{Addr: ":8085"}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no code in callback")
			http.Error(w, "No code received", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body><h1>Authorization successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)
		codeChan <- code
	})

	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Generate the authorization URL
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Println("\n╔════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Google Calendar Authorization               ║")
	fmt.Println("╠════════════════════════════════════════════════════════════════╣")
	fmt.Println("║ Please visit the following URL to authorize this application:  ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════╝")
	fmt.Printf("\n%s\n\n", authURL)

	// Try to open the URL in the browser
	if err := browser.OpenURL(authURL); err != nil {
		fmt.Println("Could not open browser automatically. Please open the URL manually.")
	}

	fmt.Println("Waiting for authorization...")

	// Wait for the callback or an error
	var code string
	select {
	case code = <-codeChan:
		// Got the code
	case err := <-errChan:
		server.Shutdown(ctx)
		return nil, err
	case <-ctx.Done():
		server.Shutdown(ctx)
		return nil, ctx.Err()
	}

	// Shutdown the server
	server.Shutdown(ctx)

	// Exchange the authorization code for a token
	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("unable to exchange code for token: %w", err)
	}

	return token, nil
}

// ClearToken removes the stored token from the keyring
func ClearToken() error {
	return keyring.DeleteToken()
}
