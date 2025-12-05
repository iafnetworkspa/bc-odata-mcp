# Test script for bc-odata-mcp server
# This script tests the MCP server by sending JSON-RPC requests

# Load environment variables
if (Test-Path "setup-bc-env.ps1") {
    . .\setup-bc-env.ps1
    Write-Host "Environment variables loaded from setup-bc-env.ps1" -ForegroundColor Green
} else {
    Write-Host "Warning: setup-bc-env.ps1 not found. Make sure environment variables are set." -ForegroundColor Yellow
}

# Check if executable exists
if (-not (Test-Path "bc-odata-mcp.exe")) {
    Write-Host "Error: bc-odata-mcp.exe not found. Run 'go build -o bc-odata-mcp.exe ./cmd/server' first." -ForegroundColor Red
    exit 1
}

Write-Host "`n=== Testing MCP Server ===" -ForegroundColor Cyan
Write-Host "Server executable: bc-odata-mcp.exe" -ForegroundColor Green
Write-Host ""

# Test 1: Initialize request
Write-Host "Test 1: Initialize request..." -ForegroundColor Yellow
$initRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "initialize"
    params = @{
        protocolVersion = "2024-11-05"
        capabilities = @{}
        clientInfo = @{
            name = "test-client"
            version = "1.0.0"
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request: $initRequest" -ForegroundColor Gray
$initRequest | .\bc-odata-mcp.exe
Write-Host ""

# Test 2: List tools
Write-Host "Test 2: List available tools..." -ForegroundColor Yellow
$toolsRequest = @{
    jsonrpc = "2.0"
    id = 2
    method = "tools/list"
} | ConvertTo-Json -Depth 10

Write-Host "Request: $toolsRequest" -ForegroundColor Gray
$toolsRequest | .\bc-odata-mcp.exe
Write-Host ""

# Test 3: Query OData endpoint (simple query)
Write-Host "Test 3: Query OData endpoint (ODV_List with top 5)..." -ForegroundColor Yellow
$queryRequest = @{
    jsonrpc = "2.0"
    id = 3
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            top = 5
            select = "No,Document_Date,Amount"
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request: $queryRequest" -ForegroundColor Gray
$queryRequest | .\bc-odata-mcp.exe
Write-Host ""

Write-Host "=== Tests completed ===" -ForegroundColor Cyan
Write-Host "`nNote: The server communicates via stdin/stdout using JSON-RPC." -ForegroundColor Gray
Write-Host "For interactive testing, you can pipe JSON requests directly to the executable." -ForegroundColor Gray

