package bc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// Auth handles Business Central OAuth 2.0 authentication
type Auth struct {
	config      Config
	httpClient  *http.Client
	token       string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// Config holds Business Central API configuration
type Config struct {
	GrantType    string
	ClientID     string
	ClientSecret string
	ScopeAPI     string
	TokenURL     string
	ContentType  string
	BasePath     string
	TenantID     string
	Environment  string
	Company      string
	APITimeout   int
}

// NewAuth creates a new Business Central authentication handler
func NewAuth(cfg Config) *Auth {
	timeout := cfg.APITimeout
	if timeout == 0 {
		timeout = 90
	}
	return &Auth{
		config: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// GetToken retrieves or refreshes the OAuth token
func (a *Auth) GetToken() (string, error) {
	a.mu.RLock()
	// Check if we have a valid token (with 5 minute safety margin)
	if a.token != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		token := a.token
		a.mu.RUnlock()
		log.Debug().Msg("Using cached OAuth token")
		return token, nil
	}
	a.mu.RUnlock()

	// Need to get a new token
	log.Info().Msg("Fetching new OAuth token from Business Central")
	return a.refreshToken()
}

// refreshToken forces a token refresh (thread-safe)
func (a *Auth) refreshToken() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Double-check after acquiring write lock
	if a.token != "" && time.Now().Before(a.tokenExpiry.Add(-5*time.Minute)) {
		log.Debug().Msg("Token was refreshed by another goroutine, using cached token")
		return a.token, nil
	}

	token, err := a.fetchToken()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch OAuth token")
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}

	a.token = token.AccessToken
	a.tokenExpiry = time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)

	log.Info().
		Time("expires_at", a.tokenExpiry).
		Int("expires_in_seconds", token.ExpiresIn).
		Msg("Successfully obtained OAuth token")

	return a.token, nil
}

// InvalidateToken invalidates the current token (e.g., after receiving 401)
func (a *Auth) InvalidateToken() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.token != "" {
		log.Warn().Msg("Invalidating expired OAuth token")
		a.token = ""
		a.tokenExpiry = time.Time{}
	}
}

// fetchToken makes the OAuth token request
func (a *Auth) fetchToken() (*TokenResponse, error) {
	log.Debug().
		Str("token_url", a.config.TokenURL).
		Str("grant_type", a.config.GrantType).
		Str("client_id", a.config.ClientID).
		Str("scope", a.config.ScopeAPI).
		Msg("Preparing OAuth token request")

	// Prepare form data
	data := url.Values{}
	data.Set("grant_type", a.config.GrantType)
	data.Set("client_id", a.config.ClientID)
	data.Set("client_secret", a.config.ClientSecret)
	data.Set("scope", a.config.ScopeAPI)

	req, err := http.NewRequest("POST", a.config.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create HTTP request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", a.config.ContentType)

	log.Debug().Msg("Sending OAuth token request")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("HTTP request failed")
		return nil, fmt.Errorf("failed to make token request: %w", err)
	}
	defer resp.Body.Close()

	log.Debug().Int("status_code", resp.StatusCode).Msg("Received token response")

	if resp.StatusCode != http.StatusOK {
		log.Error().
			Int("status_code", resp.StatusCode).
			Str("status", resp.Status).
			Msg("Token request failed with non-OK status")
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		log.Error().Err(err).Msg("Failed to decode token response")
		return nil, fmt.Errorf("failed to decode token response: %w", err)
	}

	log.Debug().Msg("Successfully decoded token response")
	return &tokenResp, nil
}

