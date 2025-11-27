#!/bin/bash
set -e

echo "=== Building Custom Docker Images for PostgreSQL HA Cluster ==="

# Build Patroni PostgreSQL image
echo "Building iaas-patroni-postgres:17..."
cd docker/patroni
docker build -t iaas-patroni-postgres:17 .
cd ../..

# Build HAProxy image
echo "Building iaas-haproxy:latest..."
cd docker/haproxy
docker build -t iaas-haproxy:latest .
cd ../..

# Build etcd image
echo "Building iaas-etcd:v3.5.11..."
cd docker/etcd
docker build -t iaas-etcd:v3.5.11 .
cd ../..

echo "=== All images built successfully! ==="
docker images | grep iaas
