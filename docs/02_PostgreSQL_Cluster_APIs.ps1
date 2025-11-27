# ============================================
# POSTGRESQL CLUSTER APIs
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
# 1. CREATE POSTGRESQL HA CLUSTER
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "1. CREATE POSTGRESQL HA CLUSTER" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createClusterBody = @{
    cluster_name = "demo-cluster"
    node_count = 3
    postgres_version = "17"
    postgres_password = "SecurePassword123!"
    cpu_per_node = 1
    memory_per_node = 1024
    storage_per_node = 10
    replication_mode = "async"
    namespace = "demo"
    enable_backup = $true
    backup_retention = 7
} | ConvertTo-Json

$createResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster" `
    -Method POST `
    -Headers $headers `
    -Body $createClusterBody `
    -ContentType "application/json"

Write-Host "[OK] Cluster Created:" -ForegroundColor Green
$createResponse | ConvertTo-Json -Depth 10

$clusterID = $createResponse.cluster_id
Write-Host "`nCluster ID: $clusterID`n" -ForegroundColor Cyan

# Wait for cluster to be ready
Write-Host "Waiting 15 seconds for cluster initialization..." -ForegroundColor Yellow
Start-Sleep -Seconds 15

# ============================================
# 2. GET CLUSTER INFO
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "2. GET CLUSTER INFO" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$clusterInfo = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Cluster Info:" -ForegroundColor Green
$clusterInfo | ConvertTo-Json -Depth 10

# ============================================
# 3. GET CLUSTER STATS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "3. GET CLUSTER STATS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$clusterStats = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/stats" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Cluster Stats:" -ForegroundColor Green
$clusterStats | ConvertTo-Json -Depth 10

# ============================================
# 4. GET REPLICATION STATUS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "4. GET REPLICATION STATUS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$replicationStatus = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/replication" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Replication Status:" -ForegroundColor Green
$replicationStatus | ConvertTo-Json -Depth 10

# ============================================
# 5. GET CLUSTER ENDPOINTS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "5. GET CLUSTER ENDPOINTS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$endpoints = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/endpoints" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Cluster Endpoints:" -ForegroundColor Green
$endpoints | ConvertTo-Json -Depth 10

# ============================================
# 6. CREATE DATABASE
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "6. CREATE DATABASE" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createDBBody = @{
    name = "demo_app_db"
    owner = "postgres"
    encoding = "UTF8"
} | ConvertTo-Json

$createDBResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/databases" `
    -Method POST `
    -Headers $headers `
    -Body $createDBBody `
    -ContentType "application/json"

Write-Host "[OK] Database Created:" -ForegroundColor Green
$createDBResponse | ConvertTo-Json -Depth 10

# ============================================
# 7. LIST DATABASES
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "7. LIST DATABASES" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$databases = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/databases" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Databases:" -ForegroundColor Green
$databases | ConvertTo-Json -Depth 10

# ============================================
# 8. CREATE USER
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "8. CREATE USER" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createUserBody = @{
    username = "app_user"
    password = "app_pass_123"
    database = "demo_app_db"
    privileges = @("SELECT", "INSERT", "UPDATE", "DELETE")
} | ConvertTo-Json

$createUserResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/users" `
    -Method POST `
    -Headers $headers `
    -Body $createUserBody `
    -ContentType "application/json"

Write-Host "[OK] User Created:" -ForegroundColor Green
$createUserResponse | ConvertTo-Json -Depth 10

# ============================================
# 9. LIST USERS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "9. LIST USERS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$users = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/users" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Users:" -ForegroundColor Green
$users | ConvertTo-Json -Depth 10

# ============================================
# 10. SCALE CLUSTER
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "10. SCALE CLUSTER (3 -> 5 nodes)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$scaleBody = @{
    node_count = 5
} | ConvertTo-Json

$scaleResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/scale" `
    -Method POST `
    -Headers $headers `
    -Body $scaleBody `
    -ContentType "application/json"

Write-Host "[OK] Cluster Scaled:" -ForegroundColor Green
$scaleResponse | ConvertTo-Json -Depth 10

# ============================================
# 11. TRIGGER BACKUP
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "11. TRIGGER BACKUP" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$backupBody = @{
    type = "full"
} | ConvertTo-Json

$backupResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/backup" `
    -Method POST `
    -Headers $headers `
    -Body $backupBody `
    -ContentType "application/json"

Write-Host "[OK] Backup Triggered:" -ForegroundColor Green
$backupResponse | ConvertTo-Json -Depth 10

# ============================================
# 12. LIST BACKUPS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "12. LIST BACKUPS" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$backups = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/backups" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Backups:" -ForegroundColor Green
$backups | ConvertTo-Json -Depth 10

# ============================================
# 13. RESTART CLUSTER
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "13. RESTART CLUSTER" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$restartResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID/restart" `
    -Method POST `
    -Headers $headers

Write-Host "[OK] Cluster Restarted:" -ForegroundColor Green
$restartResponse | ConvertTo-Json -Depth 10

Write-Host "`n" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "[SUCCESS] ALL POSTGRESQL CLUSTER APIs TESTED!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "Cluster ID: $clusterID" -ForegroundColor Cyan
Write-Host "To delete cluster: Invoke-RestMethod -Uri `"$ProvisioningBaseURL/api/v1/postgres/cluster/$clusterID`" -Method DELETE -Headers @{Authorization='Bearer $global:AuthToken'}" -ForegroundColor Yellow
