# Test with specific order 000092235
. .\setup-bc-env.ps1

Write-Host "`n=== Testing order 000092235 ===" -ForegroundColor Cyan

# Test 1: Query with filter
Write-Host "`n1. Testing bc_odata_query with filter..." -ForegroundColor Yellow
$queryRequest = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            filter = "No eq '000092235'"
            top = 1
        }
    }
} | ConvertTo-Json -Depth 10

$queryResponse = $queryRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($queryResponse) {
    if ($queryResponse.error) {
        Write-Host "ERROR:" -ForegroundColor Red
        Write-Host "  Code: $($queryResponse.error.code)" -ForegroundColor Red
        Write-Host "  Message: $($queryResponse.error.message)" -ForegroundColor Red
        if ($queryResponse.error.data) {
            Write-Host "  Details: $($queryResponse.error.data)" -ForegroundColor DarkRed
        }
    } else {
        $content = $queryResponse.result.content[0].text | ConvertFrom-Json
        Write-Host "SUCCESS: Found $($content.count) results" -ForegroundColor Green
        if ($content.count -gt 0) {
            Write-Host "Order details:" -ForegroundColor Cyan
            $order = $content.results[0]
            Write-Host "  No: $($order.No)" -ForegroundColor Gray
            Write-Host "  Document_Date: $($order.Document_Date)" -ForegroundColor Gray
            Write-Host "  Amount: $($order.Amount)" -ForegroundColor Gray
            Write-Host "  Status: $($order.Status)" -ForegroundColor Gray
        } else {
            Write-Host "No results found (order might not exist)" -ForegroundColor Yellow
        }
    }
}

# Test 2: Get entity by key
Write-Host "`n2. Testing bc_odata_get_entity..." -ForegroundColor Yellow
$getEntityRequest = @{
    jsonrpc = "2.0"
    id = 2
    method = "tools/call"
    params = @{
        name = "bc_odata_get_entity"
        arguments = @{
            endpoint = "ODV_List"
            key = "000092235"
        }
    }
} | ConvertTo-Json -Depth 10

$getEntityResponse = $getEntityRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($getEntityResponse) {
    if ($getEntityResponse.error) {
        Write-Host "ERROR:" -ForegroundColor Red
        Write-Host "  Code: $($getEntityResponse.error.code)" -ForegroundColor Red
        Write-Host "  Message: $($getEntityResponse.error.message)" -ForegroundColor Red
        if ($getEntityResponse.error.data) {
            Write-Host "  Details: $($getEntityResponse.error.data)" -ForegroundColor DarkRed
        }
    } else {
        $entityContent = $getEntityResponse.result.content[0].text | ConvertFrom-Json
        Write-Host "SUCCESS: Entity retrieved" -ForegroundColor Green
        Write-Host "Order No: $($entityContent.No)" -ForegroundColor Cyan
    }
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

