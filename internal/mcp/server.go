package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/iafnetworkspa/bc-odata-mcp/internal/bc"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Server represents the MCP server
type Server struct {
	client *bc.Client
	auth   *bc.Auth
	config bc.Config
}

// NewServer creates a new MCP server instance
func NewServer(cfg bc.Config) (*Server, error) {
	// Initialize logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	auth := bc.NewAuth(cfg)
	client := bc.NewClient(cfg, auth)

	return &Server{
		client: client,
		auth:   auth,
		config: cfg,
	}, nil
}

// Run starts the MCP server and handles JSON-RPC requests
func (s *Server) Run() error {
	// Start handling requests
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var request JSONRPCRequest
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode request: %w", err)
		}

		response := s.handleRequest(&request)
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("failed to encode response: %w", err)
		}
	}

	return nil
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(request *JSONRPCRequest) *JSONRPCResponse {
	ctx := context.Background()

	switch request.Method {
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolCall(ctx, request)
	case "initialize":
		return s.handleInitialize(request)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(request *JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: ToolCapabilities{
					ListChanged: true,
				},
			},
			ServerInfo: ServerInfo{
				Name:    "bc-odata-mcp",
				Version: "1.0.0",
			},
		},
	}
}

// handleToolsList returns the list of available tools
func (s *Server) handleToolsList(request *JSONRPCRequest) *JSONRPCResponse {
	tools := []Tool{
		{
			Name:        "bc_odata_query",
			Description: "Execute an OData query against Business Central API. Supports filtering, sorting, and pagination.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path (e.g., 'ODV_List', 'BI_Invoices', 'Customers')",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "OData $filter expression (e.g., \"No eq '12345'\")",
					},
					"select": map[string]interface{}{
						"type":        "string",
						"description": "OData $select expression to specify which fields to return",
					},
					"orderby": map[string]interface{}{
						"type":        "string",
						"description": "OData $orderby expression (e.g., 'Document_Date desc')",
					},
					"top": map[string]interface{}{
						"type":        "integer",
						"description": "OData $top expression to limit the number of results",
					},
					"skip": map[string]interface{}{
						"type":        "integer",
						"description": "OData $skip expression to skip a number of results",
					},
					"paginate": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to automatically paginate through all results (default: false)",
						"default":     false,
					},
				},
				Required: []string{"endpoint"},
			},
		},
		{
			Name:        "bc_odata_get_entity",
			Description: "Get a specific entity by its key from Business Central API.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path (e.g., 'ODV_List', 'BI_Invoices')",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key value of the entity to retrieve (e.g., order number, invoice number)",
					},
				},
				Required: []string{"endpoint", "key"},
			},
		},
		{
			Name:        "bc_odata_count",
			Description: "Get the count of entities matching a filter from Business Central API.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path (e.g., 'ODV_List', 'BI_Invoices')",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "OData $filter expression (e.g., \"No eq '12345'\")",
					},
				},
				Required: []string{"endpoint"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: ToolsListResult{
			Tools: tools,
		},
	}
}

// handleToolCall executes a tool call
func (s *Server) handleToolCall(ctx context.Context, request *JSONRPCRequest) *JSONRPCResponse {
	var params ToolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	switch params.Name {
	case "bc_odata_query":
		return s.handleODataQuery(ctx, request.ID, params.Arguments)
	case "bc_odata_get_entity":
		return s.handleGetEntity(ctx, request.ID, params.Arguments)
	case "bc_odata_count":
		return s.handleCount(ctx, request.ID, params.Arguments)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Tool not found",
			},
		}
	}
}

// handleODataQuery handles OData query requests
func (s *Server) handleODataQuery(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	endpoint, ok := args["endpoint"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: endpoint is required",
			},
		}
	}

	// Build OData query string
	queryParts := []string{}

	if filter, ok := args["filter"].(string); ok && filter != "" {
		queryParts = append(queryParts, "$filter="+filter)
	}

	if selectFields, ok := args["select"].(string); ok && selectFields != "" {
		queryParts = append(queryParts, "$select="+selectFields)
	}

	if orderby, ok := args["orderby"].(string); ok && orderby != "" {
		queryParts = append(queryParts, "$orderby="+orderby)
	}

	if top, ok := args["top"].(float64); ok && top > 0 {
		queryParts = append(queryParts, fmt.Sprintf("$top=%.0f", top))
	}

	if skip, ok := args["skip"].(float64); ok && skip > 0 {
		queryParts = append(queryParts, fmt.Sprintf("$skip=%.0f", skip))
	}

	queryString := ""
	if len(queryParts) > 0 {
		queryString = "?" + fmt.Sprintf("%s", queryParts[0])
		for i := 1; i < len(queryParts); i++ {
			queryString += "&" + queryParts[i]
		}
	}

	fullEndpoint := endpoint + queryString

	// Check if pagination is requested
	paginate := false
	if p, ok := args["paginate"].(bool); ok {
		paginate = p
	}

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, paginate)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Query execution failed",
				Data:    err.Error(),
			},
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"results": results,
		"count":  len(results),
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		},
	}
}

// handleGetEntity handles getting a specific entity by key
func (s *Server) handleGetEntity(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	endpoint, ok := args["endpoint"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: endpoint is required",
			},
		}
	}

	key, ok := args["key"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: key is required",
			},
		}
	}

	// Build endpoint with key
	fullEndpoint := fmt.Sprintf("%s('%s')", endpoint, key)

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, false)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Entity retrieval failed",
				Data:    err.Error(),
			},
		}
	}

	if len(results) == 0 {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32001,
				Message: "Entity not found",
			},
		}
	}

	resultJSON, _ := json.Marshal(results[0])

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		},
	}
}

// handleCount handles count requests
func (s *Server) handleCount(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	endpoint, ok := args["endpoint"].(string)
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: endpoint is required",
			},
		}
	}

	// Build OData query string with $count
	queryString := "?$count=true"
	if filter, ok := args["filter"].(string); ok && filter != "" {
		queryString += "&$filter=" + filter
	}

	fullEndpoint := endpoint + queryString

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, false)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Count query failed",
				Data:    err.Error(),
			},
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"count": len(results),
	})

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: ToolCallResult{
			Content: []Content{
				{
					Type: "text",
					Text: string(resultJSON),
				},
			},
		},
	}
}

