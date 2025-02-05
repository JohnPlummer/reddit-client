package reddit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const tokenURL = "https://www.reddit.com/api/v1/access_token"

type Auth struct {
	ClientID     string
	ClientSecret string
	Token        string
	ExpiresAt    time.Time
}

// Authenticate with app-only authentication (client credentials flow)
func (a *Auth) Authenticate() error {
	data := url.Values{}
	data.Set("grant_type", "client_credentials") // Use app-only auth

	req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	req.SetBasicAuth(a.ClientID, a.ClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "golang:reddit-client:v1.0 (by /u/yourusername)")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Debugging: Print Reddit API response
	body, _ := io.ReadAll(resp.Body)
	fmt.Println("Reddit API Response:", string(body))

	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to authenticate with Reddit")
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if token, ok := result["access_token"].(string); ok {
		a.Token = token
		a.ExpiresAt = time.Now().Add(time.Duration(result["expires_in"].(float64)) * time.Second)
		return nil
	}

	return errors.New("invalid response from Reddit API")
}
