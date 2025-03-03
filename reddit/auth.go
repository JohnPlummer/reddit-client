package reddit

import (
	"context"
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

// Auth represents the authentication configuration
type Auth struct {
	ClientID     string
	ClientSecret string
	Token        string
	ExpiresAt    time.Time
	userAgent    string
	client       *http.Client
	timeout      time.Duration
}

// IsTokenExpired checks if the current token is expired or about to expire
func (a *Auth) IsTokenExpired() bool {
	return time.Now().Add(time.Minute).After(a.ExpiresAt)
}

// Authenticate with app-only authentication (client credentials flow)
func (a *Auth) Authenticate(ctx context.Context) error {
	slog.InfoContext(ctx, "authenticating with Reddit")

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		slog.ErrorContext(ctx, "failed to create auth request", "error", err)
		return fmt.Errorf("creating auth request: %w", err)
	}

	req.SetBasicAuth(a.ClientID, a.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", a.userAgent)

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("making auth request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return NewAPIError(resp, body)
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

	slog.DebugContext(ctx, "authentication successful",
		"expires_in", tokenResp.ExpiresIn,
		"expires_at", a.ExpiresAt,
	)

	return nil
}

// EnsureValidToken checks if the token is expired and refreshes if necessary
func (a *Auth) EnsureValidToken(ctx context.Context) error {
	if a.IsTokenExpired() {
		slog.DebugContext(ctx, "token expired, refreshing")
		return a.Authenticate(ctx)
	}
	return nil
}

// NewAuth creates a new Auth instance with the provided credentials
func NewAuth(clientID, clientSecret string, opts ...AuthOption) (*Auth, error) {
	if clientID == "" {
		return nil, ErrMissingCredentials
	}
	if clientSecret == "" {
		return nil, ErrMissingCredentials
	}

	auth := &Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		timeout:      10 * time.Second,
		userAgent:    "golang:reddit-client:v1.0",
	}

	// Apply options
	for _, opt := range opts {
		opt(auth)
	}

	// Create default client if none was set by options
	if auth.client == nil {
		auth.client = &http.Client{
			Timeout: auth.timeout,
		}
	}

	slog.Debug("creating new auth client", "auth", auth)

	return auth, nil
}

// String returns a string representation of the Auth struct, safely handling sensitive data
func (a *Auth) String() string {
	if a == nil {
		return "Auth<nil>"
	}

	// Only obfuscate sensitive data (client secret and token)
	clientSecret := a.ClientSecret
	if len(clientSecret) > 4 {
		clientSecret = clientSecret[:4] + "..."
	}

	token := a.Token
	if len(token) > 4 {
		token = token[:4] + "..."
	}

	return fmt.Sprintf("Auth{ClientID: %q, ClientSecret: %q, Token: %q, ExpiresAt: %v, UserAgent: %q, Timeout: %v}",
		a.ClientID, // Show full client ID as it's public
		clientSecret,
		token,
		a.ExpiresAt,
		a.userAgent,
		a.timeout,
	)
}
