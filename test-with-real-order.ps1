# Test with a real order number from the sample data
. .\setup-bc-env.ps1

Write-Host "`n=== Testing with real order number ===" -ForegroundColor Cyan

# First get a real order number
Write-Host "Getting sample orders..." -ForegroundColor Yellow
$sampleRequest = @{
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

$sampleResponse = $sampleRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($sampleResponse -and -not $sampleResponse.error) {
    $sampleContent = $sampleResponse.result.content[0].text | ConvertFrom-Json
    if ($sampleContent.count -gt 0) {
        $realOrderNo = $sampleContent.results[0].No
        Write-Host "Found order: $realOrderNo" -ForegroundColor Green
        
        # Now test filtering with this real order number
        Write-Host "`nTesting filter with real order number: $realOrderNo" -ForegroundColor Yellow
        $filterRequest = @{
            jsonrpc = "2.0"
            id = 2
            method = "tools/call"
            params = @{
                name = "bc_odata_query"
                arguments = @{
                    endpoint = "ODV_List"
                    filter = "No eq '$realOrderNo'"
                    top = 1
                }
            }
        } | ConvertTo-Json -Depth 10
        
        $filterResponse = $filterRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue
        
        if ($filterResponse) {
            if ($filterResponse.error) {
                Write-Host "ERROR:" -ForegroundColor Red
                Write-Host ($filterResponse | ConvertTo-Json -Depth 10) -ForegroundColor Red
            } else {
                $filterContent = $filterResponse.result.content[0].text | ConvertFrom-Json
                Write-Host "SUCCESS: Found $($filterContent.count) results" -ForegroundColor Green
                if ($filterContent.count -gt 0) {
                    Write-Host "Order No: $($filterContent.results[0].No)" -ForegroundColor Green
                }
            }
        }
    }
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

