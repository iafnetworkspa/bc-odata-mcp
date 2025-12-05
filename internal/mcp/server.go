package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

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
		var rawRequest json.RawMessage
		if err := decoder.Decode(&rawRequest); err != nil {
			if err == io.EOF {
				break
			}
			// For parse errors, try to extract ID from raw JSON if possible
			// Otherwise, don't send a response (Cursor doesn't accept null ID)
			var temp map[string]interface{}
			if json.Unmarshal(rawRequest, &temp) == nil {
				if id, ok := temp["id"]; ok && id != nil {
					parseError := &JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      id,
						Error: &JSONRPCError{
							Code:    -32700,
							Message: "Parse error",
							Data:    err.Error(),
						},
					}
					_ = encoder.Encode(parseError)
				}
			}
			continue
		}

		var request JSONRPCRequest
		if err := json.Unmarshal(rawRequest, &request); err != nil {
			// Try to extract ID from raw request
			var temp map[string]interface{}
			if json.Unmarshal(rawRequest, &temp) == nil {
				if id, ok := temp["id"]; ok && id != nil {
					parseError := &JSONRPCResponse{
						JSONRPC: "2.0",
						ID:      id,
						Error: &JSONRPCError{
							Code:    -32700,
							Message: "Parse error",
							Data:    err.Error(),
						},
					}
					_ = encoder.Encode(parseError)
				}
			}
			continue
		}

		// Validate request
		if request.JSONRPC != "2.0" {
			if request.ID != nil {
				response := &JSONRPCResponse{
					JSONRPC: "2.0",
					ID:      request.ID,
					Error: &JSONRPCError{
						Code:    -32600,
						Message: "Invalid Request",
						Data:    "jsonrpc must be '2.0'",
					},
				}
				_ = encoder.Encode(response)
			}
			continue
		}

		// Handle notifications (requests without ID) - don't send response
		if request.ID == nil {
			// This is a notification, process it but don't send a response
			s.handleRequest(&request)
			continue
		}

		response := s.handleRequest(&request)

		// Only send response if it's not nil and has a valid ID
		if response != nil && response.ID != nil {
			if err := encoder.Encode(response); err != nil {
				return fmt.Errorf("failed to encode response: %w", err)
			}
		}
	}

	return nil
}

// handleRequest processes a JSON-RPC request
func (s *Server) handleRequest(request *JSONRPCRequest) *JSONRPCResponse {
	ctx := context.Background()

	// Validate method is present
	if request.Method == "" {
		// Only return error if this is a request (has ID), not a notification
		if request.ID != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &JSONRPCError{
					Code:    -32600,
					Message: "Invalid Request",
					Data:    "method is required",
				},
			}
		}
		return nil
	}

	switch request.Method {
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolCall(ctx, request)
	case "initialize":
		return s.handleInitialize(request)
	case "initialized":
		// This is a notification, return nil to indicate no response needed
		return nil
	default:
		// Only return error if this is a request (has ID), not a notification
		if request.ID != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      request.ID,
				Error: &JSONRPCError{
					Code:    -32601,
					Message: "Method not found",
				},
			}
		}
		return nil
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
					"expand": map[string]interface{}{
						"type":        "string",
						"description": "OData $expand expression to include related entities (e.g., 'Customer,Items')",
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
		{
			Name:        "bc_odata_list_endpoints",
			Description: "List all available OData endpoints in Business Central. This helps discover available entities and APIs.",
			InputSchema: ToolInputSchema{
				Type:       "object",
				Properties: map[string]interface{}{},
			},
		},
		{
			Name:        "bc_odata_get_metadata",
			Description: "Get OData metadata for a specific endpoint. Returns entity structure, properties, and relationships.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path (e.g., 'ODV_List', 'BI_Invoices'). Leave empty to get all metadata.",
					},
				},
			},
		},
		{
			Name:        "bc_odata_aggregate",
			Description: "Perform aggregations on OData endpoints. Supports sum, avg, min, max, count with optional grouping.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path (e.g., 'ODV_List', 'BI_Invoices')",
					},
					"aggregate": map[string]interface{}{
						"type":        "string",
						"description": "Aggregation expression (e.g., 'Amount with sum as TotalAmount,Amount with avg as AvgAmount')",
					},
					"groupby": map[string]interface{}{
						"type":        "string",
						"description": "Fields to group by (e.g., 'Document_Type,Status')",
					},
					"filter": map[string]interface{}{
						"type":        "string",
						"description": "OData $filter expression to filter data before aggregation",
					},
				},
				Required: []string{"endpoint", "aggregate"},
			},
		},
		{
			Name:        "bc_odata_create",
			Description: "Create a new entity in Business Central. Supports POST operations for writable endpoints.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path where to create the entity",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Entity data as key-value pairs",
					},
				},
				Required: []string{"endpoint", "data"},
			},
		},
		{
			Name:        "bc_odata_update",
			Description: "Update an existing entity in Business Central. Supports PATCH operations.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key value of the entity to update",
					},
					"data": map[string]interface{}{
						"type":        "object",
						"description": "Fields to update as key-value pairs",
					},
					"etag": map[string]interface{}{
						"type":        "string",
						"description": "ETag for optimistic concurrency control (optional)",
					},
				},
				Required: []string{"endpoint", "key", "data"},
			},
		},
		{
			Name:        "bc_odata_delete",
			Description: "Delete an entity from Business Central. Supports DELETE operations.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"endpoint": map[string]interface{}{
						"type":        "string",
						"description": "OData endpoint path",
					},
					"key": map[string]interface{}{
						"type":        "string",
						"description": "The key value of the entity to delete",
					},
				},
				Required: []string{"endpoint", "key"},
			},
		},
		{
			Name:        "bc_odata_check_order_status",
			Description: "Intelligently check the status of a sales order. First checks ODV_List (if found, order is not invoiced). If not found in ODV_List, checks BI_Invoices or SalesInvoices by order_no (if found, order is invoiced). If not found in either, the order may be cancelled or the order number may be incorrect.",
			InputSchema: ToolInputSchema{
				Type: "object",
				Properties: map[string]interface{}{
					"order_no": map[string]interface{}{
						"type":        "string",
						"description": "The sales order number to check",
					},
				},
				Required: []string{"order_no"},
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
	case "bc_odata_list_endpoints":
		return s.handleListEndpoints(ctx, request.ID, params.Arguments)
	case "bc_odata_get_metadata":
		return s.handleGetMetadata(ctx, request.ID, params.Arguments)
	case "bc_odata_aggregate":
		return s.handleAggregate(ctx, request.ID, params.Arguments)
	case "bc_odata_create":
		return s.handleCreate(ctx, request.ID, params.Arguments)
	case "bc_odata_update":
		return s.handleUpdate(ctx, request.ID, params.Arguments)
	case "bc_odata_delete":
		return s.handleDelete(ctx, request.ID, params.Arguments)
	case "bc_odata_check_order_status":
		return s.handleCheckOrderStatus(ctx, request.ID, params.Arguments)
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

	// Build OData query string with proper URL encoding
	queryParams := url.Values{}

	if filter, ok := args["filter"].(string); ok && filter != "" {
		queryParams.Set("$filter", filter)
	}

	if selectFields, ok := args["select"].(string); ok && selectFields != "" {
		queryParams.Set("$select", selectFields)
	}

	if orderby, ok := args["orderby"].(string); ok && orderby != "" {
		queryParams.Set("$orderby", orderby)
	}

	if top, ok := args["top"].(float64); ok && top > 0 {
		queryParams.Set("$top", fmt.Sprintf("%.0f", top))
	}

	if skip, ok := args["skip"].(float64); ok && skip > 0 {
		queryParams.Set("$skip", fmt.Sprintf("%.0f", skip))
	}

	if expand, ok := args["expand"].(string); ok && expand != "" {
		queryParams.Set("$expand", expand)
	}

	queryString := queryParams.Encode()
	fullEndpoint := endpoint
	if queryString != "" {
		fullEndpoint = endpoint + "?" + queryString
	}

	// Check if pagination is requested
	// If $top is specified, don't use automatic pagination (respect the limit)
	paginate := false
	hasTop := false
	if top, ok := args["top"].(float64); ok && top > 0 {
		hasTop = true
	}

	// Only use pagination if explicitly requested AND no $top limit is set
	if p, ok := args["paginate"].(bool); ok && p && !hasTop {
		paginate = p
	}

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, paginate)
	if err != nil {
		// Provide more descriptive error message
		errorMsg := fmt.Sprintf("Failed to execute OData query on endpoint '%s': %s", endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Query execution failed",
				Data:    errorMsg,
			},
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"results": results,
		"count":   len(results),
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

	// For endpoints with composite keys (like ODV_List), we use $filter instead of key syntax
	// This is more reliable for Business Central endpoints
	// Build OData query string with proper URL encoding
	queryParams := url.Values{}
	// Escape single quotes in the key value for OData filter
	escapedKey := strings.ReplaceAll(key, "'", "''")
	queryParams.Set("$filter", fmt.Sprintf("No eq '%s'", escapedKey))
	queryParams.Set("$top", "1")
	queryString := queryParams.Encode()
	fullEndpoint := endpoint + "?" + queryString

	// Execute query using filter (more reliable for Business Central endpoints with composite keys)
	results, err := s.client.Query(ctx, fullEndpoint, false)
	if err != nil {
		// Provide more descriptive error message
		errorMsg := fmt.Sprintf("Failed to retrieve entity '%s' from endpoint '%s': %s", key, endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Entity retrieval failed",
				Data:    errorMsg,
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
				Data:    fmt.Sprintf("No entity found with key '%s' in endpoint '%s'", key, endpoint),
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

	// Build OData query string with $count using proper URL encoding
	queryParams := url.Values{}
	queryParams.Set("$count", "true")
	if filter, ok := args["filter"].(string); ok && filter != "" {
		queryParams.Set("$filter", filter)
	}

	queryString := queryParams.Encode()
	fullEndpoint := endpoint + "?" + queryString

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, false)
	if err != nil {
		// Provide more descriptive error message
		errorMsg := fmt.Sprintf("Failed to count entities on endpoint '%s': %s", endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Count query failed",
				Data:    errorMsg,
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

// handleListEndpoints lists all available OData endpoints
func (s *Server) handleListEndpoints(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	// Business Central OData v4 structure:
	// - Root endpoint returns Company info
	// - To get entity list, we need to parse $metadata XML or try common endpoints
	// - Common endpoints in BC: ODV_List, Customers, Items, SalesOrders, PurchaseOrders, etc.

	// First, try to get metadata which contains all entity definitions
	// Metadata is at tenant/environment level, not company level
	// We'll need to construct the metadata URL differently

	// For now, return a list of common Business Central endpoints
	// and suggest using get_metadata to discover more
	commonEndpoints := []string{
		"ODV_List",
		"Customers",
		"Items",
		"SalesOrders",
		"PurchaseOrders",
		"SalesInvoices",
		"PurchaseInvoices",
		"SalesQuotes",
		"PurchaseQuotes",
		"SalesCreditMemos",
		"PurchaseCreditMemos",
		"SalesShipments",
		"PurchaseReceipts",
		"Vendors",
		"Employees",
		"GLAccounts",
		"Journals",
		"JournalLines",
		"BI_Invoices",
		"BI_Customers",
		"BI_Items",
		"BI_Vendors",
		"BI_GLAccounts",
		"BI_SalesOrders",
		"BI_PurchaseOrders",
	}

	// Try to get root endpoint to see what we get
	resp, err := s.client.Get(ctx, "")
	if err != nil {
		// If root fails, just return common endpoints
		resultJSON, _ := json.Marshal(map[string]interface{}{
			"endpoints": commonEndpoints,
			"count":     len(commonEndpoints),
			"note":      "Common Business Central endpoints. Use bc_odata_get_metadata to discover all available endpoints.",
			"error":     fmt.Sprintf("Could not query root endpoint: %s", err.Error()),
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		resultJSON, _ := json.Marshal(map[string]interface{}{
			"endpoints": commonEndpoints,
			"count":     len(commonEndpoints),
			"note":      "Common Business Central endpoints. Use bc_odata_get_metadata to discover all available endpoints.",
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

	var rootResponse map[string]interface{}
	_ = json.Unmarshal(body, &rootResponse) // Ignore error, root response is optional

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"endpoints":     commonEndpoints,
		"count":         len(commonEndpoints),
		"root_response": rootResponse,
		"note":          "Common Business Central endpoints. Use bc_odata_get_metadata to discover all available endpoints and their structure.",
		"discovery_tip": "Try querying endpoints with $top=1 to see their structure, or use bc_odata_get_metadata for complete schema information.",
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

// handleGetMetadata retrieves OData metadata for endpoints
func (s *Server) handleGetMetadata(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	// Business Central $metadata is at tenant/environment level, not company level
	// The baseURL includes company, so we need to construct metadata URL differently
	// Metadata URL format: {base}/v2.0/{tenant}/{environment}/ODataV4/$metadata

	// Try to get metadata - if it fails, try to get structure from a sample query
	endpoint := "$metadata"

	resp, err := s.client.Get(ctx, endpoint)
	if err != nil {
		// If metadata endpoint fails, try to get structure from a sample query
		// Query a known endpoint with $top=1 to infer structure
		sampleEndpoint := "ODV_List"
		if ep, ok := args["endpoint"].(string); ok && ep != "" {
			sampleEndpoint = ep
		}

		// Get sample data to infer structure
		results, queryErr := s.client.Query(ctx, sampleEndpoint+"?$top=1", false)
		if queryErr != nil {
			errorMsg := fmt.Sprintf("Failed to retrieve metadata and sample query also failed. Metadata error: %s, Query error: %s", err.Error(), queryErr.Error())
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      id,
				Error: &JSONRPCError{
					Code:    -32000,
					Message: "Failed to get metadata",
					Data:    errorMsg,
				},
			}
		}

		// Return inferred structure from sample
		var sampleFields []string
		if len(results) > 0 {
			for key := range results[0] {
				sampleFields = append(sampleFields, key)
			}
		}

		var sampleRecord interface{}
		if len(results) > 0 {
			sampleRecord = results[0]
		}

		resultJSON, _ := json.Marshal(map[string]interface{}{
			"endpoint":      sampleEndpoint,
			"metadata_note": "Could not access $metadata endpoint (may require tenant-level access). Showing inferred structure from sample query.",
			"sample_fields": sampleFields,
			"field_count":   len(sampleFields),
			"sample_record": sampleRecord,
			"tip":           "Use bc_odata_query with $top=1 on any endpoint to see its structure. Metadata endpoint may require different authentication scope.",
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
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to read metadata response: %s", err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Failed to read response",
				Data:    errorMsg,
			},
		}
	}

	// Metadata is typically XML, but we'll return it as text
	// The LLM can parse it or we could add XML parsing later
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"metadata":     string(body),
		"content_type": resp.Header.Get("Content-Type"),
		"size_bytes":   len(body),
		"note":         "Metadata is in XML format. Contains all entity type definitions, properties, and relationships.",
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

// handleAggregate performs aggregations on OData endpoints
func (s *Server) handleAggregate(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
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

	aggregate, ok := args["aggregate"].(string)
	if !ok || aggregate == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: aggregate is required",
			},
		}
	}

	// Build OData query string with $apply for aggregations using proper URL encoding
	queryParams := url.Values{}

	// Build $apply expression
	applyParts := []string{}
	if groupby, ok := args["groupby"].(string); ok && groupby != "" {
		applyParts = append(applyParts, fmt.Sprintf("groupby((%s))", groupby))
	}
	applyParts = append(applyParts, fmt.Sprintf("aggregate(%s)", aggregate))

	queryParams.Set("$apply", strings.Join(applyParts, "/"))

	if filter, ok := args["filter"].(string); ok && filter != "" {
		queryParams.Set("$filter", filter)
	}

	queryString := queryParams.Encode()
	fullEndpoint := endpoint + "?" + queryString

	// Execute query
	results, err := s.client.Query(ctx, fullEndpoint, false)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to execute aggregation on endpoint '%s': %s", endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Aggregation failed",
				Data:    errorMsg,
			},
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"results": results,
		"count":   len(results),
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

// handleCreate creates a new entity
func (s *Server) handleCreate(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
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

	data, ok := args["data"].(map[string]interface{})
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: data is required and must be an object",
			},
		}
	}

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: failed to serialize data",
				Data:    err.Error(),
			},
		}
	}

	// Create entity using POST
	result, err := s.client.Post(ctx, endpoint, jsonData)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to create entity in endpoint '%s': %s", endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Create operation failed",
				Data:    errorMsg,
			},
		}
	}

	resultJSON, _ := json.Marshal(result)
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

// handleUpdate updates an existing entity
func (s *Server) handleUpdate(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
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

	data, ok := args["data"].(map[string]interface{})
	if !ok {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: data is required and must be an object",
			},
		}
	}

	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: failed to serialize data",
				Data:    err.Error(),
			},
		}
	}

	// Build endpoint with key
	fullEndpoint := fmt.Sprintf("%s('%s')", endpoint, key)

	// Get ETag if provided for optimistic concurrency
	var etag string
	if e, ok := args["etag"].(string); ok {
		etag = e
	}

	// Update entity using PATCH
	result, err := s.client.Patch(ctx, fullEndpoint, jsonData, etag)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to update entity '%s' in endpoint '%s': %s", key, endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Update operation failed",
				Data:    errorMsg,
			},
		}
	}

	resultJSON, _ := json.Marshal(result)
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

// handleDelete deletes an entity
func (s *Server) handleDelete(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
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

	// Delete entity using DELETE
	err := s.client.Delete(ctx, fullEndpoint)
	if err != nil {
		errorMsg := fmt.Sprintf("Failed to delete entity '%s' from endpoint '%s': %s", key, endpoint, err.Error())
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32000,
				Message: "Delete operation failed",
				Data:    errorMsg,
			},
		}
	}

	resultJSON, _ := json.Marshal(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Entity '%s' deleted successfully from endpoint '%s'", key, endpoint),
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

// handleCheckOrderStatus intelligently checks the status of a sales order
// Logic:
// 1. Check ODV_List first - if found, order is NOT invoiced
// 2. If not found in ODV_List, check BI_Invoices or SalesInvoices by order_no
//   - If found in invoices, order IS invoiced
//   - If not found in either, order may be cancelled or order number is incorrect
func (s *Server) handleCheckOrderStatus(ctx context.Context, id interface{}, args map[string]interface{}) *JSONRPCResponse {
	orderNo, ok := args["order_no"].(string)
	if !ok || orderNo == "" {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      id,
			Error: &JSONRPCError{
				Code:    -32602,
				Message: "Invalid params: order_no is required",
			},
		}
	}

	// Step 1: Check ODV_List first
	// If order is found in ODV_List, it means it's NOT invoiced
	queryParams := url.Values{}
	escapedOrderNo := strings.ReplaceAll(orderNo, "'", "''")
	queryParams.Set("$filter", fmt.Sprintf("No eq '%s'", escapedOrderNo))
	queryParams.Set("$top", "1")
	odvEndpoint := "ODV_List?" + queryParams.Encode()

	odvResults, err := s.client.Query(ctx, odvEndpoint, false)
	if err != nil {
		// If ODV_List query fails, we'll still try invoices
		// Log the error but continue
		log.Error().Err(err).Str("order_no", orderNo).Msg("Error querying ODV_List, will try invoices")
	}

	if len(odvResults) > 0 {
		// Order found in ODV_List - it's NOT invoiced
		resultJSON, _ := json.Marshal(map[string]interface{}{
			"order_no":     orderNo,
			"status":       "not_invoiced",
			"status_label": "Ordine non fatturato",
			"found_in":     "ODV_List",
			"message":      fmt.Sprintf("L'ordine %s è stato trovato in ODV_List, quindi NON è ancora stato fatturato.", orderNo),
			"order_data":   odvResults[0],
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

	// Step 2: Order not found in ODV_List, check invoices
	// Try BI_Invoices first (Business Intelligence endpoint)
	queryParams = url.Values{}
	queryParams.Set("$filter", fmt.Sprintf("Order_No eq '%s'", escapedOrderNo))
	queryParams.Set("$top", "1")
	invoiceEndpoint := "BI_Invoices?" + queryParams.Encode()

	invoiceResults, err := s.client.Query(ctx, invoiceEndpoint, false)
	if err != nil || len(invoiceResults) == 0 {
		// If BI_Invoices fails or returns nothing, try SalesInvoices
		queryParams = url.Values{}
		queryParams.Set("$filter", fmt.Sprintf("Order_No eq '%s'", escapedOrderNo))
		queryParams.Set("$top", "1")
		invoiceEndpoint = "SalesInvoices?" + queryParams.Encode()
		invoiceResults, _ = s.client.Query(ctx, invoiceEndpoint, false)
	}

	if len(invoiceResults) > 0 {
		// Order found in invoices - it IS invoiced
		resultJSON, _ := json.Marshal(map[string]interface{}{
			"order_no":     orderNo,
			"status":       "invoiced",
			"status_label": "Ordine fatturato",
			"found_in":     "Invoices",
			"message":      fmt.Sprintf("L'ordine %s non è stato trovato in ODV_List ma è stato trovato nelle fatture, quindi È STATO FATTURATO.", orderNo),
			"invoice_data": invoiceResults[0],
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

	// Step 3: Order not found in either ODV_List or invoices
	// It may be cancelled, or the order number is incorrect/partial
	resultJSON, _ := json.Marshal(map[string]interface{}{
		"order_no":     orderNo,
		"status":       "not_found",
		"status_label": "Ordine non trovato",
		"found_in":     "none",
		"message":      fmt.Sprintf("L'ordine %s non è stato trovato né in ODV_List né nelle fatture. Potrebbe essere stato cancellato, oppure il numero ordine potrebbe essere errato o parziale.", orderNo),
		"suggestions": []string{
			"Verificare che il numero ordine sia corretto e completo",
			"Controllare se l'ordine è stato cancellato",
			"Verificare se l'ordine esiste in altri endpoint (es. SalesOrders)",
		},
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
