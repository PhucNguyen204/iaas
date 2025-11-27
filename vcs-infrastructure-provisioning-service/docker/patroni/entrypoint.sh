#!/bin/bash
set -e

# Function to check etcd connectivity
check_etcd_health() {
  local etcd_hostname=$1
  local etcd_port=$2
  
  # Method 1: Try etcdctl health check (most reliable)
  if command -v etcdctl &> /dev/null; then
    if ETCDCTL_API=3 etcdctl --endpoints="http://${etcd_hostname}:${etcd_port}" endpoint health &>/dev/null 2>&1; then
      return 0
    fi
  fi
  
  # Method 2: Try HTTP health endpoint
  if command -v curl &> /dev/null; then
    local http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 2 "http://${etcd_hostname}:${etcd_port}/health" 2>/dev/null)
    if [ "$http_code" = "200" ] || [ "$http_code" = "503" ]; then
      # 200 = healthy, 503 = unhealthy but service is responding
      return 0
    fi
  fi
  
  # Method 3: Try TCP connection
  if timeout 2 bash -c "cat < /dev/null > /dev/tcp/${etcd_hostname}/${etcd_port}" 2>/dev/null; then
    return 0
  fi
  
  return 1
}

# Function to wait for etcd to be ready
wait_for_etcd() {
  local etcd_host=${ETCD_HOST:-etcd-1:2379}
  local max_attempts=90  # 3 minutes total (90 * 2s)
  local attempt=1
  local wait_interval=2
  
  echo "=========================================="
  echo "Waiting for etcd to be ready..."
  echo "etcd endpoint: ${etcd_host}"
  echo "=========================================="
  
  # Extract host and port
  local etcd_hostname=$(echo $etcd_host | cut -d: -f1)
  local etcd_port=$(echo $etcd_host | cut -d: -f2)
  
  if [ -z "$etcd_hostname" ] || [ -z "$etcd_port" ]; then
    echo "⚠ WARNING: Invalid ETCD_HOST format: ${etcd_host}"
    echo "Expected format: hostname:port (e.g., etcd-1:2379)"
    return 0
  fi
  
  while [ $attempt -le $max_attempts ]; do
    if check_etcd_health "$etcd_hostname" "$etcd_port"; then
      echo "✓ etcd is ready and healthy at ${etcd_host}!"
      echo "Proceeding with Patroni startup..."
      return 0
    fi
    
    if [ $((attempt % 5)) -eq 0 ]; then
      echo "[$attempt/$max_attempts] etcd not ready yet, waiting ${wait_interval}s..."
    fi
    
    if [ $attempt -lt $max_attempts ]; then
      sleep $wait_interval
      # Exponential backoff after 20 attempts
      if [ $attempt -gt 20 ] && [ $attempt -le 40 ]; then
        wait_interval=3
      elif [ $attempt -gt 40 ]; then
        wait_interval=5
      fi
    fi
    
    attempt=$((attempt + 1))
  done
  
  echo "⚠ WARNING: etcd health check failed after ${max_attempts} attempts"
  echo "This may indicate etcd cluster is not fully initialized"
  echo "Patroni will attempt to connect anyway and will retry automatically"
  echo "If Patroni fails to start, check etcd cluster status"
  return 0
}

# Fix permissions on mounted volumes FIRST
echo "Ensuring correct permissions on data directories..."
if [ -d "/data" ]; then
  chmod 700 /data 2>/dev/null || true
fi
if [ -d "${PGDATA}" ]; then
  chmod 700 "${PGDATA}" 2>/dev/null || true
fi

# Create pgpass file with wildcard for replication
echo "Creating pgpass file for replication..."
mkdir -p /opt/secretpg
cat > /opt/secretpg/pgpass <<PGPASS_EOF
*:5432:*:replicator:${REPLICATION_PASSWORD:-replicator_pass}
*:5432:*:postgres:${POSTGRES_PASSWORD}
PGPASS_EOF
chmod 600 /opt/secretpg/pgpass
export PGPASSFILE=/opt/secretpg/pgpass

# Wait for etcd before proceeding
wait_for_etcd

# Generate PgBackRest config if enabled
if [ "${PGBACKREST_ENABLED}" = "true" ]; then
cat > /etc/pgbackrest.conf <<PGBACKREST_EOF
[global]
repo1-path=/pgbackrest/repo
repo1-retention-full=${PGBACKREST_RETENTION:-7}
log-level-console=info
log-level-file=debug
log-path=/pgbackrest/log
process-max=${PGBACKREST_PROCESS_MAX:-2}

[${SCOPE}]
pg1-path=${PGDATA}
pg1-port=5432
PGBACKREST_EOF
fi

# Generate patroni.yml from template and environment variables
cat > /etc/patroni/patroni.yml <<EOF
scope: "${SCOPE}"
namespace: "${NAMESPACE:-percona_lab}"
name: "${PATRONI_NAME}"

restapi:
  listen: 0.0.0.0:8008
  connect_address: ${PATRONI_NAME}:8008

etcd3:
  host: ${ETCD_HOST}
  protocol: http

bootstrap:
  dcs:
    ttl: 30
    loop_wait: 10
    retry_timeout: 10
    maximum_lag_on_failover: 1048576
    postgresql:
      use_pg_rewind: true
      use_slots: true
      parameters:
        max_connections: ${MAX_CONNECTIONS:-100}
        shared_buffers: ${SHARED_BUFFERS:-256MB}
        wal_level: replica
        hot_standby: on
        wal_keep_segments: 10
        max_wal_senders: ${MAX_WAL_SENDERS:-20}
        max_replication_slots: ${MAX_REPLICATION_SLOTS:-20}
        wal_log_hints: on
        logging_collector: 'on'
        max_wal_size: '10GB'
        archive_mode: ${ARCHIVE_MODE:-on}
        archive_timeout: ${ARCHIVE_TIMEOUT:-600s}
        archive_command: '${ARCHIVE_COMMAND:-test ! -f /pgbackrest/archive/%f && pgbackrest --stanza=${SCOPE} archive-push %p}'
        timezone: 'Asia/Ho_Chi_Minh'
        work_mem: 4MB
        synchronous_commit: ${SYNCHRONOUS_COMMIT:-local}
        synchronous_standby_names: '${SYNCHRONOUS_STANDBY_NAMES:-}'
      pg_hba:
        - local   all             all                                 peer
        - host    replication     replicator   127.0.0.1/32           trust
        - host    replication     replicator   0.0.0.0/0              scram-sha-256
        - host    all             all          0.0.0.0/0              scram-sha-256
      recovery_conf:
        restore_command: '${RESTORE_COMMAND:-pgbackrest --stanza=${SCOPE} archive-get %f "%p"}'

  initdb:
    - encoding: UTF8
    - data-checksums

  users:
    postgres:
      password: ${POSTGRES_PASSWORD}
      options:
        - createrole
        - createdb
    replicator:
      password: ${REPLICATION_PASSWORD}
      options:
        - replication

postgresql:
  cluster_name: ${SCOPE}
  listen: 0.0.0.0:5432
  connect_address: ${PATRONI_NAME}:5432
  data_dir: ${PGDATA}
  bin_dir: /usr/lib/postgresql/17/bin
  pgpass: /opt/secretpg/pgpass
  authentication:
    replication:
      username: replicator
      password: ${REPLICATION_PASSWORD}
    superuser:
      username: postgres
      password: ${POSTGRES_PASSWORD}
  parameters:
    unix_socket_directories: '/var/run/postgresql'
  create_replica_methods:
    - custom_basebackup
  custom_basebackup:
    command: /tmp/custom_basebackup.sh
    keep_data: True
    no_params: True
  callbacks:
    on_role_change: /tmp/on_role_change.sh
    on_start: /tmp/fix_permissions.sh
    on_reload: /tmp/on_reload.sh
  pg_hba:
    - local   all             all                                 peer
    - host    replication     replicator   127.0.0.1/32           trust
    - host    replication     replicator   0.0.0.0/0              scram-sha-256
    - host    all             all          0.0.0.0/0              scram-sha-256

watchdog:
  mode: ${WATCHDOG_MODE:-off}

tags:
  nofailover: ${NOFAILOVER:-false}
  noloadbalance: ${NOLOADBALANCE:-false}
  clonefrom: ${CLONEFROM:-false}
  nosync: ${NOSYNC:-false}
EOF

echo "Starting Patroni with config:"
cat /etc/patroni/patroni.yml

# Create custom basebackup script for replicas
cat > /tmp/custom_basebackup.sh <<'BASEBACKUP_EOF'
#!/bin/bash
set -e

# Get connection info from Patroni environment
PGHOST="${PATRONI_SCOPE}-0.${PATRONI_SCOPE}.default.svc.cluster.local"
PGPORT=5432
PGUSER="replicator"

echo "Creating replica using pg_basebackup..."
echo "Master: ${PATRONI_MASTER_CONNECT_ADDRESS}"

# Clean data directory if exists
if [ -d "${PGDATA}" ]; then
  echo "Cleaning ${PGDATA}..."
  rm -rf "${PGDATA}"/*
fi

# Run pg_basebackup
/usr/lib/postgresql/17/bin/pg_basebackup \
  -h "${PATRONI_MASTER_HOST:-patroni-node-1}" \
  -p 5432 \
  -U replicator \
  -D "${PGDATA}" \
  -X stream \
  -c fast \
  -R \
  -v

# Fix permissions after basebackup
echo "Fixing permissions on ${PGDATA}..."
chmod 700 "${PGDATA}"
chmod 600 "${PGDATA}"/*.conf 2>/dev/null || true

echo "Basebackup completed successfully"
exit 0
BASEBACKUP_EOF

chmod +x /tmp/custom_basebackup.sh

# Create callback scripts
cat > /tmp/fix_permissions.sh <<'CALLBACK_EOF'
#!/bin/bash
# Fix permissions BEFORE PostgreSQL starts
if [ -d "${PGDATA}" ]; then
  chmod 700 "${PGDATA}"
  chown -R postgres:postgres "${PGDATA}" 2>/dev/null || true
  echo "Fixed permissions on ${PGDATA} to 700"
fi
exit 0
CALLBACK_EOF

cat > /tmp/on_role_change.sh <<'CALLBACK_EOF'
#!/bin/bash
# Role change callback
echo "Role changed to: $1"
exit 0
CALLBACK_EOF

cat > /tmp/on_reload.sh <<'CALLBACK_EOF'
#!/bin/bash
# Reload callback
echo "Configuration reloaded"
exit 0
CALLBACK_EOF

chmod +x /tmp/fix_permissions.sh /tmp/on_role_change.sh /tmp/on_reload.sh

# Initialize PgBackRest stanza if this is the first node
if [ "${PGBACKREST_ENABLED}" = "true" ] && [ "${IS_LEADER}" = "true" ]; then
  echo "Waiting for PostgreSQL to start..."
  sleep 10
  echo "Creating PgBackRest stanza: ${SCOPE}"
  pgbackrest --stanza=${SCOPE} --log-level-console=info stanza-create || true
fi

# Start Patroni
exec patroni /etc/patroni/patroni.yml
