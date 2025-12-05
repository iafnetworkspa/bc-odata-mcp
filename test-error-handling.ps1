# Test error handling with invalid filter
. .\setup-bc-env.ps1

Write-Host "`n=== Testing error handling ===" -ForegroundColor Cyan

# Test with invalid filter (number without quotes on string field)
Write-Host "Testing invalid filter (number on string field)..." -ForegroundColor Yellow
$errorRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            filter = "No eq 5700003963"
            top = 1
        }
    }
} | ConvertTo-Json -Depth 10

$errorResponse = $errorRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($errorResponse) {
    if ($errorResponse.error) {
        Write-Host "ERROR (expected):" -ForegroundColor Yellow
        Write-Host "  Code: $($errorResponse.error.code)" -ForegroundColor Gray
        Write-Host "  Message: $($errorResponse.error.message)" -ForegroundColor Gray
        if ($errorResponse.error.data) {
            Write-Host "  Details: $($errorResponse.error.data)" -ForegroundColor Gray
        }
    } else {
        Write-Host "UNEXPECTED SUCCESS (should have failed)" -ForegroundColor Red
    }
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

