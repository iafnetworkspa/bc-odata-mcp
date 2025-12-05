# Test script for filtering ODV_List by order number
# Load environment and test the specific query that's failing

. .\setup-bc-env.ps1

Write-Host "`n=== Testing ODV_List filter query ===" -ForegroundColor Cyan
Write-Host ""

# Test: Query ODV_List with filter for order 5700003963
$request = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            filter = "No eq '5700003963'"
            top = 1
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request:" -ForegroundColor Yellow
Write-Host $request -ForegroundColor Gray
Write-Host "`nResponse:" -ForegroundColor Yellow

$response = $request | .\bc-odata-mcp.exe 2>&1

# Try to parse JSON response
$jsonResponse = $response | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($jsonResponse) {
    if ($jsonResponse.error) {
        Write-Host "ERROR:" -ForegroundColor Red
        Write-Host ($jsonResponse | ConvertTo-Json -Depth 10) -ForegroundColor Red
    } else {
        Write-Host "SUCCESS:" -ForegroundColor Green
        Write-Host ($jsonResponse | ConvertTo-Json -Depth 10) -ForegroundColor Green
    }
} else {
    Write-Host "Raw response:" -ForegroundColor Yellow
    $response | ForEach-Object { Write-Host $_ }
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

