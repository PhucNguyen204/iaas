#!/bin/bash
set -e

echo "=========================================="
echo "Starting etcd node: ${ETCD_NAME}"
echo "Initial cluster: ${ETCD_INITIAL_CLUSTER}"
echo "Initial cluster state: ${ETCD_INITIAL_CLUSTER_STATE}"
echo "=========================================="

# Save env vars to local variables
NAME=${ETCD_NAME}
DATA_DIR="/etcd-data"  # Match volume mount path
CLUSTER_TOKEN=${ETCD_INITIAL_CLUSTER_TOKEN:-pgcluster}
CLUSTER_STATE=${ETCD_INITIAL_CLUSTER_STATE:-new}
INITIAL_CLUSTER=${ETCD_INITIAL_CLUSTER}
ADVERTISE_PEER=${ETCD_INITIAL_ADVERTISE_PEER_URLS}
LISTEN_PEER=${ETCD_LISTEN_PEER_URLS}
ADVERTISE_CLIENT=${ETCD_ADVERTISE_CLIENT_URLS}
LISTEN_CLIENT=${ETCD_LISTEN_CLIENT_URLS}

# Ensure data directory exists and has correct permissions
mkdir -p ${DATA_DIR}
chmod 700 ${DATA_DIR} 2>/dev/null || true

# Validate required environment variables
if [ -z "$NAME" ]; then
  echo "ERROR: ETCD_NAME is required"
  exit 1
fi

if [ -z "$INITIAL_CLUSTER" ]; then
  echo "ERROR: ETCD_INITIAL_CLUSTER is required"
  exit 1
fi

echo "Data directory: ${DATA_DIR}"
echo "Cluster token: ${CLUSTER_TOKEN}"
echo "Cluster state: ${CLUSTER_STATE}"

# Unset all ETCD_ environment variables to avoid conflicts
unset ETCD_NAME ETCD_INITIAL_CLUSTER_TOKEN ETCD_INITIAL_CLUSTER_STATE
unset ETCD_INITIAL_CLUSTER ETCD_INITIAL_ADVERTISE_PEER_URLS
unset ETCD_LISTEN_PEER_URLS ETCD_ADVERTISE_CLIENT_URLS ETCD_LISTEN_CLIENT_URLS

echo "Starting etcd process..."
exec etcd \
  --name ${NAME} \
  --data-dir ${DATA_DIR} \
  --initial-cluster-token ${CLUSTER_TOKEN} \
  --initial-cluster-state ${CLUSTER_STATE} \
  --initial-cluster ${INITIAL_CLUSTER} \
  --initial-advertise-peer-urls ${ADVERTISE_PEER} \
  --listen-peer-urls ${LISTEN_PEER} \
  --advertise-client-urls ${ADVERTISE_CLIENT} \
  --listen-client-urls ${LISTEN_CLIENT} \
  --heartbeat-interval 1000 \
  --election-timeout 10000
