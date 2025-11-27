#!/bin/sh
set -e

# Generate HAProxy config with node list from environment
if [ -n "$PATRONI_NODES" ]; then
  echo "Configuring HAProxy with Patroni nodes: $PATRONI_NODES"
  
  # Create base config
  cat > /tmp/haproxy.cfg <<'EOF'
global
    maxconn 100
    log stdout format raw local0

defaults
    log global
    mode tcp
    retries 2
    timeout client 30m
    timeout connect 4s
    timeout server 30m
    timeout check 5s

listen stats
    mode http
    bind *:7000
    stats enable
    stats uri /
    stats refresh 5s

listen primary
    bind *:5000
    option httpchk GET /primary
    http-check expect status 200
    default-server inter 3s fall 3 rise 2
EOF
  
  # Add primary servers
  for node in $(echo $PATRONI_NODES | tr ',' ' '); do
    echo "    server ${node} ${node}:5432 maxconn 100 check port 8008" >> /tmp/haproxy.cfg
  done
  
  # Add standbys section
  cat >> /tmp/haproxy.cfg <<'EOF'

listen standbys
    balance roundrobin
    bind *:5001
    option httpchk GET /replica
    http-check expect status 200
    default-server inter 3s fall 3 rise 2
EOF
  
  # Add standby servers
  for node in $(echo $PATRONI_NODES | tr ',' ' '); do
    echo "    server ${node} ${node}:5432 maxconn 100 check port 8008" >> /tmp/haproxy.cfg
  done
  
  # Use the generated config
  set -- haproxy -f /tmp/haproxy.cfg
fi

# Start HAProxy
exec "$@"
