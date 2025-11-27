# PowerShell script to build custom Docker images for PostgreSQL HA Cluster
Write-Host "=== Building Custom Docker Images for PostgreSQL HA Cluster ===" -ForegroundColor Green

# Build Patroni PostgreSQL image
Write-Host "`nBuilding iaas-patroni-postgres:17..." -ForegroundColor Cyan
Set-Location docker\patroni
docker build -t iaas-patroni-postgres:17 .
Set-Location ..\..

# Build HAProxy image
Write-Host "`nBuilding iaas-haproxy:latest..." -ForegroundColor Cyan
Set-Location docker\haproxy
docker build -t iaas-haproxy:latest .
Set-Location ..\..

# Build etcd image
Write-Host "`nBuilding iaas-etcd:v3.5.11..." -ForegroundColor Cyan
Set-Location docker\etcd
docker build -t iaas-etcd:v3.5.11 .
Set-Location ..\..

Write-Host "`n=== All images built successfully! ===" -ForegroundColor Green
docker images | Select-String iaas
