# Test script for bc_odata_list_endpoints tool
# This script tests the MCP server by sending a request to list all OData endpoints

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

Write-Host "`n=== Testing bc_odata_list_endpoints ===" -ForegroundColor Cyan
Write-Host ""

# Test: List endpoints
Write-Host "Test: List all available OData endpoints..." -ForegroundColor Yellow
$listEndpointsRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_list_endpoints"
        arguments = @{}
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request:" -ForegroundColor Gray
Write-Host $listEndpointsRequest -ForegroundColor DarkGray
Write-Host "`nResponse:" -ForegroundColor Gray

$listEndpointsRequest | .\bc-odata-mcp.exe | ConvertFrom-Json | ConvertTo-Json -Depth 20

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

