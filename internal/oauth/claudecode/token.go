// Package claudecode provides OAuth token management for Claude Code subscriptions.
package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/crush/internal/oauth"
)

const (
	// OAuthTokenPrefix is the prefix for Claude Code OAuth access tokens (sk-ant-oat*).
	OAuthTokenPrefix = "sk-ant-oat"

	tokenURL = "https://platform.claude.com/v1/oauth/token"
	clientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
)

// RefreshToken exchanges a refresh token for a new access token using Anthropic's OAuth endpoint.
func RefreshToken(ctx context.Context, refreshToken string) (*oauth.Token, error) {
	body, _ := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     clientID,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to build token refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token refresh returned status %d", resp.StatusCode)
	}

	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"` // seconds
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode token refresh response: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return &oauth.Token{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresIn:    result.ExpiresIn,
		ExpiresAt:    expiresAt.Unix(),
	}, nil
}
