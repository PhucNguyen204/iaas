# ============================================
# DOCKER SERVICE APIs
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
# 1. CREATE DOCKER SERVICE (Redis)
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "1. CREATE DOCKER SERVICE (Redis)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createRedisBody = @{
    name = "app-cache-redis"
    image = "redis:7-alpine"
    cpu = 1
    memory = 512
    ports = @(
        @{ container_port = 6379; host_port = 6380 }
    )
    environment = @{
        "REDIS_PASSWORD" = "redis_pass_123"
    }
    volumes = @(
        @{ name = "redis-data"; mount_path = "/data" }
    )
    command = @("redis-server", "--requirepass", "redis_pass_123")
} | ConvertTo-Json -Depth 10

$redisResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker" `
    -Method POST `
    -Headers $headers `
    -Body $createRedisBody `
    -ContentType "application/json"

Write-Host "[OK] Redis Service Created:" -ForegroundColor Green
$redisResponse | ConvertTo-Json -Depth 10

$redisID = $redisResponse.data.id
Write-Host "`nRedis Service ID: $redisID`n" -ForegroundColor Cyan

Start-Sleep -Seconds 3

# ============================================
# 2. CREATE DOCKER SERVICE (MongoDB)
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "2. CREATE DOCKER SERVICE (MongoDB)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createMongoBody = @{
    name = "app-database-mongo"
    image = "mongo:7"
    cpu = 2
    memory = 2048
    ports = @(
        @{ container_port = 27017; host_port = 27018 }
    )
    environment = @{
        "MONGO_INITDB_ROOT_USERNAME" = "admin"
        "MONGO_INITDB_ROOT_PASSWORD" = "mongo_pass_123"
        "MONGO_INITDB_DATABASE" = "app_db"
    }
    volumes = @(
        @{ name = "mongo-data"; mount_path = "/data/db" }
        @{ name = "mongo-config"; mount_path = "/data/configdb" }
    )
} | ConvertTo-Json -Depth 10

$mongoResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker" `
    -Method POST `
    -Headers $headers `
    -Body $createMongoBody `
    -ContentType "application/json"

Write-Host "[OK] MongoDB Service Created:" -ForegroundColor Green
$mongoResponse | ConvertTo-Json -Depth 10

$mongoID = $mongoResponse.data.id
Write-Host "`nMongoDB Service ID: $mongoID`n" -ForegroundColor Cyan

Start-Sleep -Seconds 3

# ============================================
# 3. CREATE DOCKER SERVICE (RabbitMQ)
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "3. CREATE DOCKER SERVICE (RabbitMQ)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$createRabbitBody = @{
    name = "app-queue-rabbitmq"
    image = "rabbitmq:3-management-alpine"
    cpu = 1
    memory = 1024
    ports = @(
        @{ container_port = 5672; host_port = 5673 }
        @{ container_port = 15672; host_port = 15673 }
    )
    environment = @{
        "RABBITMQ_DEFAULT_USER" = "admin"
        "RABBITMQ_DEFAULT_PASS" = "rabbit_pass_123"
    }
    volumes = @(
        @{ name = "rabbitmq-data"; mount_path = "/var/lib/rabbitmq" }
    )
} | ConvertTo-Json -Depth 10

$rabbitResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker" `
    -Method POST `
    -Headers $headers `
    -Body $createRabbitBody `
    -ContentType "application/json"

Write-Host "[OK] RabbitMQ Service Created:" -ForegroundColor Green
$rabbitResponse | ConvertTo-Json -Depth 10

$rabbitID = $rabbitResponse.data.id
Write-Host "`nRabbitMQ Service ID: $rabbitID`n" -ForegroundColor Cyan

Start-Sleep -Seconds 3

# ============================================
# 4. GET DOCKER SERVICE INFO
# ============================================
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "4. GET DOCKER SERVICE INFO (Redis)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$serviceInfo = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$redisID" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Service Info:" -ForegroundColor Green
$serviceInfo | ConvertTo-Json -Depth 10

# ============================================
# 5. UPDATE ENVIRONMENT VARIABLES
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "5. UPDATE ENVIRONMENT VARIABLES" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$updateEnvBody = @{
    environment = @{
        "REDIS_PASSWORD" = "new_redis_pass_456"
        "REDIS_MAXMEMORY" = "256mb"
        "REDIS_MAXMEMORY_POLICY" = "allkeys-lru"
    }
} | ConvertTo-Json

$updateEnvResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$redisID/env" `
    -Method PUT `
    -Headers $headers `
    -Body $updateEnvBody `
    -ContentType "application/json"

Write-Host "[OK] Environment Updated:" -ForegroundColor Green
$updateEnvResponse | ConvertTo-Json -Depth 10

# ============================================
# 6. GET SERVICE LOGS
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "6. GET SERVICE LOGS (MongoDB)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$logs = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$mongoID/logs?tail=30" `
    -Method GET `
    -Headers $headers

Write-Host "[OK] Service Logs (last 30 lines):" -ForegroundColor Green
$logs | ConvertTo-Json -Depth 10

# ============================================
# 7. STOP DOCKER SERVICE
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "7. STOP DOCKER SERVICE (RabbitMQ)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$stopResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$rabbitID/stop" `
    -Method POST `
    -Headers $headers

Write-Host "[OK] Service Stopped:" -ForegroundColor Green
$stopResponse | ConvertTo-Json -Depth 10

Start-Sleep -Seconds 2

# ============================================
# 8. START DOCKER SERVICE
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "8. START DOCKER SERVICE (RabbitMQ)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$startResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$rabbitID/start" `
    -Method POST `
    -Headers $headers

Write-Host "[OK] Service Started:" -ForegroundColor Green
$startResponse | ConvertTo-Json -Depth 10

Start-Sleep -Seconds 2

# ============================================
# 9. RESTART DOCKER SERVICE
# ============================================
Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "9. RESTART DOCKER SERVICE (Redis)" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow

$restartResponse = Invoke-RestMethod -Uri "$ProvisioningBaseURL/api/v1/docker/$redisID/restart" `
    -Method POST `
    -Headers $headers

Write-Host "[OK] Service Restarted:" -ForegroundColor Green
$restartResponse | ConvertTo-Json -Depth 10

Write-Host "`n" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "[SUCCESS] ALL DOCKER SERVICE APIs TESTED!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host "Created Services:" -ForegroundColor Cyan
Write-Host "  - Redis ID: $redisID" -ForegroundColor White
Write-Host "  - MongoDB ID: $mongoID" -ForegroundColor White
Write-Host "  - RabbitMQ ID: $rabbitID" -ForegroundColor White
Write-Host "`nTo delete services:" -ForegroundColor Yellow
Write-Host "  Invoke-RestMethod -Uri `"$ProvisioningBaseURL/api/v1/docker/$redisID`" -Method DELETE -Headers @{Authorization='Bearer $global:AuthToken'}" -ForegroundColor Gray
Write-Host "  Invoke-RestMethod -Uri `"$ProvisioningBaseURL/api/v1/docker/$mongoID`" -Method DELETE -Headers @{Authorization='Bearer $global:AuthToken'}" -ForegroundColor Gray
Write-Host "  Invoke-RestMethod -Uri `"$ProvisioningBaseURL/api/v1/docker/$rabbitID`" -Method DELETE -Headers @{Authorization='Bearer $global:AuthToken'}" -ForegroundColor Gray
