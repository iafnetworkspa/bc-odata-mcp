# Test script to query a known endpoint and analyze the response
# This helps understand the structure and discover other endpoints

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

Write-Host "`n=== Testing known endpoint: ODV_List ===" -ForegroundColor Cyan
Write-Host ""

# Test: Query ODV_List with top 1 to see structure
$queryRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            top = 1
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request:" -ForegroundColor Gray
Write-Host $queryRequest -ForegroundColor DarkGray
Write-Host "`nResponse:" -ForegroundColor Gray

$response = $queryRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern "jsonrpc" -Context 0,50
$response

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

