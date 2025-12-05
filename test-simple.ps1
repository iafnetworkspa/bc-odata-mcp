# Simple test - just list tools
# Load environment and test tools/list

. .\setup-bc-env.ps1

$request = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/list"
} | ConvertTo-Json

Write-Host "Sending tools/list request..." -ForegroundColor Cyan
$request | .\bc-odata-mcp.exe

