package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/iafnetworkspa/bc-odata-mcp/internal/bc"
)

func TestServer_NewServer(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	if server == nil {
		t.Fatal("NewServer returned nil")
	}
	if server.client == nil {
		t.Fatal("NewServer client is nil")
	}
}

func TestJSONRPCRequest_Unmarshal(t *testing.T) {
	jsonData := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
	var request JSONRPCRequest
	err := json.Unmarshal([]byte(jsonData), &request)
	if err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if request.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", request.JSONRPC)
	}
	if request.ID != float64(1) {
		t.Errorf("ID = %v, want 1", request.ID)
	}
	if request.Method != "tools/list" {
		t.Errorf("Method = %v, want tools/list", request.Method)
	}
}

func TestJSONRPCResponse_Marshal(t *testing.T) {
	response := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result: ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: "test",
				},
			},
		},
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("Marshaled JSON is invalid: %s", string(data))
	}
}

func TestJSONRPCError_Marshal(t *testing.T) {
	error := &JSONRPCError{
		Code:    -32600,
		Message: "Invalid Request",
		Data:    "test data",
	}

	data, err := json.Marshal(error)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !json.Valid(data) {
		t.Errorf("Marshaled JSON is invalid: %s", string(data))
	}
}

func TestServer_handleInitialize(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}
	response := server.handleInitialize(request)

	if response == nil {
		t.Fatal("handleInitialize returned nil")
	}
	if response.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", response.JSONRPC)
	}
	if response.Result == nil {
		t.Fatal("Result is nil")
	}
}

func TestServer_handleToolsList(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	request := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "tools/list",
	}
	response := server.handleToolsList(request)

	if response == nil {
		t.Fatal("handleToolsList returned nil")
	}
	if response.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", response.JSONRPC)
	}
	if response.Result == nil {
		t.Fatal("Result is nil")
	}
}

func TestServer_handleODataQuery_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleODataQuery(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleODataQuery returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleGetEntity_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleGetEntity(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleGetEntity returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleCount_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleCount(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleCount returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleListEndpoints(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleListEndpoints(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleListEndpoints returned nil")
	}
	if response.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %v, want 2.0", response.JSONRPC)
	}
}

func TestServer_handleAggregate_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleAggregate(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleAggregate returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleCreate_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleCreate(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleCreate returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleUpdate_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleUpdate(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleUpdate returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}

func TestServer_handleDelete_InvalidParams(t *testing.T) {
	cfg := bc.Config{
		GrantType:    "client_credentials",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		ScopeAPI:     "https://api.businesscentral.dynamics.com/.default",
		TokenURL:     "https://login.microsoftonline.com/test/oauth2/v2.0/token",
		ContentType:  "application/x-www-form-urlencoded",
		BasePath:     "https://api.businesscentral.dynamics.com/v2.0",
		APITimeout:   90,
	}

	server, _ := NewServer(cfg)

	ctx := context.Background()
	response := server.handleDelete(ctx, 1, map[string]interface{}{})

	if response == nil {
		t.Fatal("handleDelete returned nil")
	}
	if response.Error == nil {
		t.Fatal("Expected error for missing endpoint")
	}
	if response.Error.Code != -32602 {
		t.Errorf("Error code = %v, want -32602", response.Error.Code)
	}
}
