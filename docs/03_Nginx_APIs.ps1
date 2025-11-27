# ============================================
# NGINX APIs
# Service: vcs-infrastructure-provisioning-service
# Base URL: http://localhost:8083
# ============================================

# Variables
$ProvisioningBaseURL = "http://localhost:8083"
$AuthBaseURL = "http://localhost:8082"

# ============================================
# 0. GET AUTH TOKEN
# ============================================
Write-Host "[AUTH] Getting authentication token..." -ForegroundColor Cyan
$loginBody = @{
    username = "admin"
    password = "password123"
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "$AuthBaseURL/auth/login" -Method POST -Body $loginBody -ContentType "application/json"
$global:AuthToken = $loginResponse.data.access_token
$headers = @{ "Authorization" = "Bearer $global:AuthToken" }
Write-Host "[OK] Token obtained`n" -ForegroundColor Green

# ============================================
# 1. CREATE NGINX INSTANCE
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "1. CREATE NGINX INSTANCE" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createNginxBody = @{
    name = "api-gateway"
    image = "nginx:alpine"
    port = 8080
    cpu = 1
    memory = 512
    domains = @("api.example.com", "www.example.com")
    upstreams = @(
        @{
            name = "backend"
            servers = @(
                @{ host = "backend1"; port = 3000; weight = 1 }
                @{ host = "backend2"; port = 3000; weight = 1 }
            )
            algorithm = "round_robin"
        }
    )
    routes = @(
        @{
            path = "/api"
            upstream = "backend"
            methods = @("GET", "POST")
        }
    )
} | ConvertTo-Json -Depth 10

$createResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx" `
    -Method POST `
    -Headers $headers `
    -Body $createNginxBody `
    -ContentType "application/json"

Write-Host "[OK] Nginx Created:" -ForegroundColor Green
$createResponse | ConvertTo-Json -Depth 10

$nginxID = $createResponse.data.id
Write-Host "`nNginx ID: $nginxID`n" -ForegroundColor Cyan

Start-Sleep -Seconds 3

# ============================================
# 2. GET NGINX INFO
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "2. GET NGINX INFO" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$nginxInfo = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Nginx Info:" -ForegroundColor Green
$nginxInfo | ConvertTo-Json -Depth 10

# ============================================
# 3. ADD DOMAIN
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "3. ADD DOMAIN" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$addDomainBody = @{
    domain = "admin.example.com"
    ssl_enabled = $false
} | ConvertTo-Json

$addDomainResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/domains" `
    -Method POST `
    -Headers $headers `
    -Body $addDomainBody `
    -ContentType "application/json"

Write-Host "[OK] Domain Added:" -ForegroundColor Green
$addDomainResponse | ConvertTo-Json -Depth 10

# ============================================
# 4. ADD ROUTE
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "4. ADD ROUTE" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$addRouteBody = @{
    path = "/admin"
    upstream = "backend"
    methods = @("GET", "POST", "PUT", "DELETE")
    rate_limit = 100
} | ConvertTo-Json

$addRouteResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/routes" `
    -Method POST `
    -Headers $headers `
    -Body $addRouteBody `
    -ContentType "application/json"

Write-Host "[OK] Route Added:" -ForegroundColor Green
$addRouteResponse | ConvertTo-Json -Depth 10

$routeID = $addRouteResponse.data.id

# ============================================
# 5. UPDATE UPSTREAMS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "5. UPDATE UPSTREAMS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$updateUpstreamsBody = @{
    upstreams = @(
        @{
            name = "backend"
            servers = @(
                @{ host = "backend1"; port = 3000; weight = 2 }
                @{ host = "backend2"; port = 3000; weight = 1 }
                @{ host = "backend3"; port = 3000; weight = 1 }
            )
            algorithm = "least_conn"
            health_check = @{
                enabled = $true
                interval = 30
                timeout = 5
            }
        }
    )
} | ConvertTo-Json -Depth 10

$updateUpstreamsResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/upstreams" `
    -Method PUT `
    -Headers $headers `
    -Body $updateUpstreamsBody `
    -ContentType "application/json"

Write-Host "[OK] Upstreams Updated:" -ForegroundColor Green
$updateUpstreamsResponse | ConvertTo-Json -Depth 10

# ============================================
# 6. GET UPSTREAMS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "6. GET UPSTREAMS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$upstreams = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/upstreams" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Upstreams:" -ForegroundColor Green
$upstreams | ConvertTo-Json -Depth 10

# ============================================
# 7. SET SECURITY POLICY
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "7. SET SECURITY POLICY" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$securityBody = @{
    enable_cors = $true
    cors_origins = @("https://app.example.com", "https://admin.example.com")
    cors_methods = @("GET", "POST", "PUT", "DELETE", "OPTIONS")
    cors_headers = @("Content-Type", "Authorization")
    enable_rate_limit = $true
    rate_limit_requests = 100
    rate_limit_window = 60
    enable_ip_whitelist = $false
    ip_whitelist = @()
} | ConvertTo-Json

$securityResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/security" `
    -Method POST `
    -Headers $headers `
    -Body $securityBody `
    -ContentType "application/json"

Write-Host "[OK] Security Policy Set:" -ForegroundColor Green
$securityResponse | ConvertTo-Json -Depth 10

# ============================================
# 8. GET SECURITY POLICY
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "8. GET SECURITY POLICY" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$security = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/security" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Security Policy:" -ForegroundColor Green
$security | ConvertTo-Json -Depth 10

# ============================================
# 9. GET STATS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "9. GET NGINX STATS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$stats = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/stats" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Nginx Stats:" -ForegroundColor Green
$stats | ConvertTo-Json -Depth 10

# ============================================
# 10. GET METRICS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "10. GET NGINX METRICS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$metrics = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/metrics" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Nginx Metrics:" -ForegroundColor Green
$metrics | ConvertTo-Json -Depth 10

# ============================================
# 11. GET LOGS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "11. GET NGINX LOGS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$logs = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/logs?tail=50" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Nginx Logs (last 50 lines):" -ForegroundColor Green
$logs | ConvertTo-Json -Depth 10

# ============================================
# 12. RESTART NGINX
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "12. RESTART NGINX" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$restartResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/nginx/$nginxID/restart" `
    -Method POST `
    -Headers $headers

Write-Host "[OK] Nginx Restarted:" -ForegroundColor Green
$restartResponse | ConvertTo-Json -Depth 10

Write-Host "`n" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "[SUCCESS] ALL NGINX APIs TESTED!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "Nginx ID: $nginxID" -ForegroundColor Cyan
Write-Host "To delete nginx: Invoke-RestMethod -Uri `"$ProvisioningBaseURL/api/v1/nginx/$nginxID`" -Method DELETE -Headers @{Authorization='Bearer $global:AuthToken'}" -ForegroundColor Yellow
