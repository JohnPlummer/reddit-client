package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	tokenURL      = "https://www.reddit.com/api/v1/access_token"
	tokenLifetime = time.Hour // Reddit tokens typically last 1 hour
)

// TokenResponse represents the Reddit OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

type Auth struct {
	ClientID     string
	ClientSecret string
	Token        string
	ExpiresAt    time.Time
}

// IsTokenExpired checks if the current token is expired or about to expire
func (a *Auth) IsTokenExpired() bool {
	return time.Now().Add(time.Minute).After(a.ExpiresAt)
}

// Authenticate with app-only authentication (client credentials flow)
func (a *Auth) Authenticate() error {
	slog.Info("authenticating with Reddit")

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		slog.Error("failed to create auth request", "error", err)
		return fmt.Errorf("creating auth request: %w", err)
	}

	req.SetBasicAuth(a.ClientID, a.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "golang:reddit-client:v1.0 (by /u/yourusername)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("making auth request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("parsing token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return fmt.Errorf("no access token in response")
	}

	a.Token = tokenResp.AccessToken
	a.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	slog.Debug("authentication successful",
		"expires_in", tokenResp.ExpiresIn,
		"expires_at", a.ExpiresAt,
	)

	return nil
}

// EnsureValidToken checks if the token is expired and refreshes if necessary
func (a *Auth) EnsureValidToken() error {
	if a.IsTokenExpired() {
		slog.Debug("token expired, refreshing")
		return a.Authenticate()
	}
	return nil
}

// Add this function
func NewAuth(clientID, clientSecret string) *Auth {
	if clientID == "" {
		slog.Error("client ID is required")
		return nil
	}
	if clientSecret == "" {
		slog.Error("client secret is required")
		return nil
	}

	slog.Debug("creating new auth client",
		"client_id", clientID,
		"client_secret", clientSecret[:4]+"...", // Only show first 4 chars of secret
	)

	return &Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
}
