# Analyze endpoint structure to identify additional features to implement
# Load environment and query ODV_List to see structure

. .\setup-bc-env.ps1

$request = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            top = 1
            select = "No,Document_Date,Amount,Status,Customer_No,Customer_Name"
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Analyzing ODV_List structure..." -ForegroundColor Cyan
$response = $request | .\bc-odata-mcp.exe 2>&1 | Select-String -Pattern "jsonrpc" -Context 0,5

# Parse and show structure
$jsonResponse = $response -join "`n" | ConvertFrom-Json
$content = $jsonResponse.result.content[0].text | ConvertFrom-Json
$results = $content.results

if ($results -and $results.Count -gt 0) {
    Write-Host "`nSample record fields:" -ForegroundColor Green
    $results[0].PSObject.Properties.Name | Sort-Object | ForEach-Object {
        $value = $results[0].$_ 
        $type = if ($value -eq $null) { "null" } 
                elseif ($value -is [string]) { "string" }
                elseif ($value -is [int]) { "int" }
                elseif ($value -is [double]) { "decimal" }
                elseif ($value -is [bool]) { "bool" }
                elseif ($value -is [datetime]) { "datetime" }
                else { $value.GetType().Name }
        Write-Host "  $_ : $type" -ForegroundColor Gray
    }
}

Write-Host "`nPotential features to implement:" -ForegroundColor Yellow
Write-Host "1. Expand relationships (e.g., Customer details, Item details)" -ForegroundColor White
Write-Host "2. Aggregations (sum, avg, min, max, count by group)" -ForegroundColor White
Write-Host "3. Advanced filters (contains, startswith, endswith, date ranges)" -ForegroundColor White
Write-Host "4. CRUD operations (Create, Update, Delete entities)" -ForegroundColor White
Write-Host "5. Batch operations (multiple operations in one request)" -ForegroundColor White
Write-Host "6. Search across multiple endpoints" -ForegroundColor White
Write-Host "7. Export to different formats (CSV, Excel)" -ForegroundColor White

