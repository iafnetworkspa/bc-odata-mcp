package bc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := Config{
		BasePath:   "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout: 90,
	}
	auth := NewAuth(cfg)

	client := NewClient(cfg, auth)
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.baseURL != cfg.BasePath {
		t.Errorf("Expected baseURL %s, got %s", cfg.BasePath, client.baseURL)
	}
	if client.httpClient.Timeout != 90*time.Second {
		t.Errorf("Expected timeout 90s, got %v", client.httpClient.Timeout)
	}
}

func TestNewClient_DefaultTimeout(t *testing.T) {
	cfg := Config{
		BasePath:   "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout: 0, // Should default to 90
	}
	auth := NewAuth(cfg)

	client := NewClient(cfg, auth)
	if client.httpClient.Timeout != 90*time.Second {
		t.Errorf("Expected default timeout 90s, got %v", client.httpClient.Timeout)
	}
}

func TestClient_Query_Success(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected Authorization header Bearer test-token, got %s", r.Header.Get("Authorization"))
		}
		odataResp := ODataResponse{
			Value: []map[string]interface{}{
				{"No": "001", "Name": "Test Item"},
				{"No": "002", "Name": "Test Item 2"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(odataResp)
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	results, err := client.Query(ctx, "/test", false)
	if err != nil {
		t.Fatalf("Query() error = %v, want nil", err)
	}
	if len(results) != 2 {
		t.Errorf("Query() returned %d results, want 2", len(results))
	}
	if results[0]["No"] != "001" {
		t.Errorf("Query() first result No = %v, want 001", results[0]["No"])
	}
}

func TestClient_Query_WithPagination(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	pageCount := 0
	// Mock OData server with pagination
	var odataServerURL string
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pageCount++
		var odataResp ODataResponse
		if pageCount == 1 {
			odataResp = ODataResponse{
				Value: []map[string]interface{}{
					{"No": "001", "Name": "Test Item 1"},
				},
				NextLink: odataServerURL + "/test?$skip=1",
			}
		} else {
			odataResp = ODataResponse{
				Value: []map[string]interface{}{
					{"No": "002", "Name": "Test Item 2"},
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(odataResp)
	}))
	odataServerURL = odataServer.URL
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	results, err := client.Query(ctx, "/test", true) // Enable pagination
	if err != nil {
		t.Fatalf("Query() error = %v, want nil", err)
	}
	if len(results) < 2 {
		t.Errorf("Query() returned %d results, want at least 2", len(results))
	}
}

func TestClient_Query_HTTPError(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server returning error
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"message":"Not Found"}}`))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	_, err := client.Query(ctx, "/test", false)
	if err == nil {
		t.Fatal("Query() error = nil, want error")
	}
}

func TestClient_Query_InvalidJSON(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server returning invalid JSON
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	_, err := client.Query(ctx, "/test", false)
	if err == nil {
		t.Fatal("Query() error = nil, want error")
	}
}

func TestClient_Get_Success(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"value":[]}`))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	resp, err := client.Get(ctx, "/test")
	if err != nil {
		t.Fatalf("Get() error = %v, want nil", err)
	}
	if resp == nil {
		t.Fatal("Get() returned nil response")
	}
	resp.Body.Close()
}

func TestClient_GetPaginated_WithTopLimit(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		odataResp := ODataResponse{
			Value: []map[string]interface{}{
				{"No": "001"},
				{"No": "002"},
				{"No": "003"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(odataResp)
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	results, err := client.GetPaginated(ctx, "/test?$top=2")
	if err != nil {
		t.Fatalf("GetPaginated() error = %v, want nil", err)
	}
	if len(results) != 2 {
		t.Errorf("GetPaginated() returned %d results, want 2", len(results))
	}
}

func TestClient_GetPaginated_NoNextLink(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	requestCount := 0
	// Mock OData server - first request returns data, second returns empty
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var odataResp ODataResponse
		if requestCount == 1 {
			odataResp = ODataResponse{
				Value: []map[string]interface{}{
					{"No": "001"},
				},
				// No NextLink
			}
		} else {
			odataResp = ODataResponse{
				Value: []map[string]interface{}{},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(odataResp)
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	results, err := client.GetPaginated(ctx, "/test")
	if err != nil {
		t.Fatalf("GetPaginated() error = %v, want nil", err)
	}
	if len(results) != 1 {
		t.Errorf("GetPaginated() returned %d results, want 1", len(results))
	}
}

func TestClient_Post_Success(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"No":"001","Name":"Test Item"}`))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	data := []byte(`{"Name":"Test Item"}`)
	result, err := client.Post(ctx, "/test", data)
	if err != nil {
		t.Fatalf("Post() error = %v, want nil", err)
	}
	if result["No"] != "001" {
		t.Errorf("Post() result No = %v, want 001", result["No"])
	}
}

func TestClient_Patch_Success(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"No":"001","Name":"Updated Item"}`))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	data := []byte(`{"Name":"Updated Item"}`)
	result, err := client.Patch(ctx, "/test('001')", data, "")
	if err != nil {
		t.Fatalf("Patch() error = %v, want nil", err)
	}
	if result["Name"] != "Updated Item" {
		t.Errorf("Patch() result Name = %v, want Updated Item", result["Name"])
	}
}

func TestClient_Patch_WithETag(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("If-Match") != "W/\"test-etag\"" {
			t.Errorf("Expected If-Match header W/\"test-etag\", got %s", r.Header.Get("If-Match"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"No":"001","Name":"Updated Item"}`))
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	data := []byte(`{"Name":"Updated Item"}`)
	result, err := client.Patch(ctx, "/test('001')", data, "W/\"test-etag\"")
	if err != nil {
		t.Fatalf("Patch() error = %v, want nil", err)
	}
	if result["Name"] != "Updated Item" {
		t.Errorf("Patch() result Name = %v, want Updated Item", result["Name"])
	}
}

func TestClient_Delete_Success(t *testing.T) {
	// Mock OAuth server
	oauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenResp := TokenResponse{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResp)
	}))
	defer oauthServer.Close()

	// Mock OData server
	odataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer odataServer.Close()

	cfg := Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     oauthServer.URL,
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     odataServer.URL,
		APITimeout:   90,
	}

	auth := NewAuth(cfg)
	client := NewClient(cfg, auth)

	ctx := context.Background()
	err := client.Delete(ctx, "/test('001')")
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
}

func TestODataResponse_JSON(t *testing.T) {
	// Test JSON marshaling/unmarshaling
	jsonData := `{"value":[{"No":"001","Name":"Test"}],"@odata.nextLink":"/next"}`
	var odataResp ODataResponse
	err := json.Unmarshal([]byte(jsonData), &odataResp)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(odataResp.Value) != 1 {
		t.Errorf("Value length = %d, want 1", len(odataResp.Value))
	}
	if odataResp.NextLink != "/next" {
		t.Errorf("NextLink = %v, want /next", odataResp.NextLink)
	}

	// Test marshaling
	marshaled, err := json.Marshal(odataResp)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !json.Valid(marshaled) {
		t.Errorf("Marshaled JSON is invalid: %s", string(marshaled))
	}
}
