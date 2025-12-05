# Test different filter formats to find the correct one
. .\setup-bc-env.ps1

Write-Host "`n=== Testing different filter formats ===" -ForegroundColor Cyan

$testCases = @(
    @{ name = "String with quotes"; filter = "No eq '5700003963'" },
    @{ name = "Number without quotes"; filter = "No eq 5700003963" },
    @{ name = "String with double quotes"; filter = "No eq `"5700003963`"" },
    @{ name = "Contains"; filter = "contains(No, '5700003963')" },
    @{ name = "Startswith"; filter = "startswith(No, '5700003963')" }
)

foreach ($testCase in $testCases) {
    Write-Host "`n--- Testing: $($testCase.name) ---" -ForegroundColor Yellow
    Write-Host "Filter: $($testCase.filter)" -ForegroundColor Gray
    
    $request = @{
        jsonrpc = "2.0"
        id = 1
        method = "tools/call"
        params = @{
            name = "bc_odata_query"
            arguments = @{
                endpoint = "ODV_List"
                filter = $testCase.filter
                top = 1
            }
        }
    } | ConvertTo-Json -Depth 10

    $response = $request | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

    if ($response) {
        if ($response.error) {
            Write-Host "ERROR: $($response.error.message)" -ForegroundColor Red
            if ($response.error.data) {
                Write-Host "Details: $($response.error.data)" -ForegroundColor DarkRed
            }
        } else {
            $content = $response.result.content[0].text | ConvertFrom-Json
            Write-Host "SUCCESS: Found $($content.count) results" -ForegroundColor Green
            if ($content.count -gt 0) {
                Write-Host "First result No: $($content.results[0].No)" -ForegroundColor Green
            }
        }
    }
}

# Also test getting a sample record to see what the No field looks like
Write-Host "`n--- Getting sample record to check No field format ---" -ForegroundColor Yellow
$sampleRequest = @{
    jsonrpc = "2.0"
    id = 2
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            top = 5
        }
    }
} | ConvertTo-Json -Depth 10

$sampleResponse = $sampleRequest | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern '\{.*\}' | ForEach-Object { $_.Matches[0].Value } | ConvertFrom-Json -ErrorAction SilentlyContinue

if ($sampleResponse -and -not $sampleResponse.error) {
    $sampleContent = $sampleResponse.result.content[0].text | ConvertFrom-Json
    if ($sampleContent.count -gt 0) {
        Write-Host "Sample No values:" -ForegroundColor Cyan
        $sampleContent.results | Select-Object -First 5 | ForEach-Object {
            Write-Host "  No: '$($_.No)' (Type: $($_.No.GetType().Name))" -ForegroundColor Gray
        }
    }
}

Write-Host "`n=== Test completed ===" -ForegroundColor Cyan

