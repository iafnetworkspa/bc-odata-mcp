package bc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAuth(t *testing.T) {
	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	if auth == nil {
		t.Fatal("NewAuth returned nil")
	}
	if auth.config.ClientID != cfg.ClientID {
		t.Errorf("Expected ClientID %s, got %s", cfg.ClientID, auth.config.ClientID)
	}
	if auth.httpClient.Timeout != 90*time.Second {
		t.Errorf("Expected timeout 90s, got %v", auth.httpClient.Timeout)
	}
}

func TestNewAuth_DefaultTimeout(t *testing.T) {
	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   0, // Should default to 90
	}

	auth := NewAuth(cfg)
	if auth.httpClient.Timeout != 90*time.Second {
		t.Errorf("Expected default timeout 90s, got %v", auth.httpClient.Timeout)
	}
}

func TestAuth_GetToken_WithValidCachedToken(t *testing.T) {
	auth := &Auth{
		token:       "cached-token",
		tokenExpiry: time.Now().Add(10 * time.Minute),
	}

	token, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v, want nil", err)
	}
	if token != "cached-token" {
		t.Errorf("GetToken() = %v, want cached-token", token)
	}
}

func TestAuth_GetToken_WithExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		tokenResp := TokenResponse{
			AccessToken: "new-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer server.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     server.URL,
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	auth.token = "old-token"
	auth.tokenExpiry = time.Now().Add(-1 * time.Hour) // Expired

	token, err := auth.GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v, want nil", err)
	}
	if token != "new-token" {
		t.Errorf("GetToken() = %v, want new-token", token)
	}
}

func TestAuth_InvalidateToken(t *testing.T) {
	auth := &Auth{
		token:       "test-token",
		tokenExpiry: time.Now().Add(10 * time.Minute),
	}

	auth.InvalidateToken()

	if auth.token != "" {
		t.Errorf("InvalidateToken() did not clear token, got %s", auth.token)
	}
	if !auth.tokenExpiry.IsZero() {
		t.Errorf("InvalidateToken() did not clear tokenExpiry")
	}
}

func TestAuth_InvalidateToken_EmptyToken(t *testing.T) {
	auth := &Auth{
		token:       "",
		tokenExpiry: time.Time{},
	}

	// Should not panic
	auth.InvalidateToken()

	if auth.token != "" {
		t.Errorf("InvalidateToken() changed empty token to %s", auth.token)
	}
}

func TestAuth_fetchToken_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
		}

		// Verify form data
		r.ParseForm()
		if r.Form.Get("grant_type") != "client_credentials" {
			t.Errorf("Expected grant_type client_credentials, got %s", r.Form.Get("grant_type"))
		}

		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       "https://api.businesscentral.dynamics.com/.default",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer server.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     server.URL,
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	tokenResp, err := auth.fetchToken()
	if err != nil {
		t.Fatalf("fetchToken() error = %v, want nil", err)
	}
	if tokenResp.AccessToken != "test-token" {
		t.Errorf("fetchToken() AccessToken = %v, want test-token", tokenResp.AccessToken)
	}
	if tokenResp.ExpiresIn != 3600 {
		t.Errorf("fetchToken() ExpiresIn = %v, want 3600", tokenResp.ExpiresIn)
	}
}

func TestAuth_fetchToken_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}))
	defer server.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     server.URL,
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	_, err := auth.fetchToken()
	if err == nil {
		t.Fatal("fetchToken() error = nil, want error")
	}
}

func TestAuth_fetchToken_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     server.URL,
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	_, err := auth.fetchToken()
	if err == nil {
		t.Fatal("fetchToken() error = nil, want error")
	}
}

func TestAuth_refreshToken_DoubleCheck(t *testing.T) {
	// Test that refreshToken properly double-checks after acquiring write lock
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "new-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer server.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     server.URL,
		ContentType:  "application/x-www-form-urlencoded",
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	auth.token = "old-token"
	auth.tokenExpiry = time.Now().Add(-1 * time.Hour) // Expired

	token, err := auth.refreshToken()
	if err != nil {
		t.Fatalf("refreshToken() error = %v, want nil", err)
	}
	if token != "new-token" {
		t.Errorf("refreshToken() = %v, want new-token", token)
	}
	if auth.token != "new-token" {
		t.Errorf("auth.token = %v, want new-token", auth.token)
	}
}

func TestTokenResponse_JSON(t *testing.T) {
	// Test JSON marshaling/unmarshaling
	jsonData := `{"access_token":"test-token","token_type":"Bearer","expires_in":3600,"scope":"test-scope"}`
	var tokenResp TokenResponse
	err := json.Unmarshal([]byte(jsonData), &tokenResp)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if tokenResp.AccessToken != "test-token" {
		t.Errorf("AccessToken = %v, want test-token", tokenResp.AccessToken)
	}
	if tokenResp.TokenType != "Bearer" {
		t.Errorf("TokenType = %v, want Bearer", tokenResp.TokenType)
	}
	if tokenResp.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %v, want 3600", tokenResp.ExpiresIn)
	}

	// Test marshaling
	marshaled, err := json.Marshal(tokenResp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !bytes.Contains(marshaled, []byte("test-token")) {
		t.Errorf("Marshaled JSON does not contain test-token: %s", string(marshaled))
	}
}


