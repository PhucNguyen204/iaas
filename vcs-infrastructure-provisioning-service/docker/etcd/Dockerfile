FROM ubuntu:22.04

# Install etcd
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install etcd v3.5.x
ENV ETCD_VERSION=v3.5.11
RUN curl -L https://github.com/etcd-io/etcd/releases/download/${ETCD_VERSION}/etcd-${ETCD_VERSION}-linux-amd64.tar.gz -o /tmp/etcd.tar.gz \
    && tar xzvf /tmp/etcd.tar.gz -C /tmp \
    && mv /tmp/etcd-${ETCD_VERSION}-linux-amd64/etcd /usr/local/bin/ \
    && mv /tmp/etcd-${ETCD_VERSION}-linux-amd64/etcdctl /usr/local/bin/ \
    && rm -rf /tmp/etcd*

# Create etcd data directory
RUN mkdir -p /var/lib/etcd && \
    chmod 700 /var/lib/etcd

# Expose etcd ports
# 2379 - Client communication
# 2380 - Peer communication
EXPOSE 2379 2380

# Copy entrypoint
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
