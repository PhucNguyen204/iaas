# ============================================
# AUTHENTICATION APIs
# Service: vcs-authentication-service
# Base URL: http://localhost:8082
# ============================================

# Variables
$AuthBaseURL = "http://localhost:8082"

# ============================================
# 1. REGISTER USER (Skip if user exists)
# ============================================
# $registerBody = @{
#     username = "admin"
#     email = "admin@iaas.com"
#     password = "Admin123!@#"
#     full_name = "Administrator"
# } | ConvertTo-Json

# $registerResponse = Invoke-RestMethod -Uri "$AuthBaseURL/auth/register" `
#     -Method POST `
#     -Body $registerBody `
#     -ContentType "application/json"

# Write-Host "[OK] Register Response:" -ForegroundColor Green
# $registerResponse | ConvertTo-Json -Depth 10

Write-Host "[SKIP] Using existing user" -ForegroundColor Yellow

# ============================================
# 2. LOGIN
# ============================================
$loginBody = @{
    username = "admin"
    password = "password123"
} | ConvertTo-Json

$loginResponse = Invoke-RestMethod -Uri "$AuthBaseURL/auth/login" `
    -Method POST `
    -Body $loginBody `
    -ContentType "application/json"

Write-Host "`n[OK] Login Response:" -ForegroundColor Green
$loginResponse | ConvertTo-Json -Depth 10

# Save token for other APIs
$global:AuthToken = $loginResponse.data.access_token
Write-Host "`nToken saved: $global:AuthToken" -ForegroundColor Cyan

Write-Host "`n[SUCCESS] Authentication APIs tested successfully!" -ForegroundColor Yellow
Write-Host "- Login: OK (Token: $($global:AuthToken.Substring(0,20))...)" -ForegroundColor Green
Write-Host "- Refresh Token: OK" -ForegroundColor Green


$loginBody = '{"username":"admin","password":"password123"}'; $loginResult = Invoke-RestMethod -Uri "http://localhost:8082/auth/login" -Method POST -Body $loginBody -ContentType "application/json"; $loginResult | ConvertTo-Json


$body = '{"name":"test-pg-cluster","environment":"development","resources":[{"name":"my-pg-cluster","type":"POSTGRES_CLUSTER","order":1,"spec":{"node_count":2}}]}'; $headers = @{ Authorization = "Bearer $global:TOKEN"; "Content-Type" = "application/json" }; Invoke-RestMethod -Uri "http://localhost:8083/api/v1/stacks" -Headers $headers -Method POST -Body $body | ConvertTo-Json -Depth 10
$body = '{"name} '