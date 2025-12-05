# Test query to Business Central
# This script tests a real OData query

. .\setup-bc-env.ps1

Write-Host "Testing OData query to Business Central..." -ForegroundColor Cyan
Write-Host ""

# Query ODV_List with top 3 results
$request = @{
    jsonrpc = "2.0"
    id = 1
    method = "tools/call"
    params = @{
        name = "bc_odata_query"
        arguments = @{
            endpoint = "ODV_List"
            top = 3
            select = "No,Document_Date,Amount"
            orderby = "Document_Date desc"
        }
    }
} | ConvertTo-Json -Depth 10

Write-Host "Request:" -ForegroundColor Yellow
Write-Host $request -ForegroundColor Gray
Write-Host ""
Write-Host "Response:" -ForegroundColor Yellow

$request | .\bc-odata-mcp.exe | ConvertFrom-Json | ConvertTo-Json -Depth 10

