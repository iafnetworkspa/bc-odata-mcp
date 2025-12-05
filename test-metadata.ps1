# Test script for bc_odata_get_metadata tool
# This script tests the MCP server by sending a request to get OData metadata

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

Write-Host "`n=== Testing bc_odata_get_metadata ===" -ForegroundColor Cyan
Write-Host ""

# Test: Get metadata
Write-Host "Test: Get OData metadata..." -ForegroundColor Yellow
$getMetadataRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_get_metadata"
        arguments = @{}
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request:" -ForegroundColor Gray
Write-Host $getMetadataRequest -ForegroundColor DarkGray
Write-Host "`nResponse (first 2000 chars):" -ForegroundColor Gray

$response = $getMetadataRequest | .\bc-odata-mcp.exe | ConvertFrom-Json
$content = $response.result.content[0].text | ConvertFrom-Json
$metadata = $content.metadata

# Show first part of metadata
if ($metadata.Length -gt 2000) {
    Write-Host $metadata.Substring(0, 2000) -ForegroundColor DarkGray
    Write-Host "`n... (truncated, total length: $($metadata.Length) bytes)" -ForegroundColor Gray
} else {
    Write-Host $metadata -ForegroundColor DarkGray
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

