FROM ubuntu:22.04

# Prevent interactive prompts
ENV DEBIAN_FRONTEND=noninteractive
ENV TZ=Asia/Ho_Chi_Minh

# Install PostgreSQL 17 + Patroni in one layer for faster build
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    gnupg2 \
    lsb-release \
    wget \
    jq \
    ca-certificates \
    python3-pip \
    python3-dev \
    python3-psycopg2 \
    etcd-client \
    && curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc | gpg --dearmor -o /usr/share/keyrings/postgresql-keyring.gpg \
    && echo "deb [signed-by=/usr/share/keyrings/postgresql-keyring.gpg] http://apt.postgresql.org/pub/repos/apt $(lsb_release -cs)-pgdg main" > /etc/apt/sources.list.d/pgdg.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        postgresql-17 \
        postgresql-client-17 \
        postgresql-contrib-17 \
    && rm -rf /var/lib/apt/lists/*

# Install Patroni and dependencies
RUN pip3 install --no-cache-dir patroni[etcd3]==3.2.2 python-etcd

# Setup PgBackRest (WAL archiving tool)
RUN apt-get update && apt-get install -y --no-install-recommends pgbackrest && \
    rm -rf /var/lib/apt/lists/*

# Create directories with proper permissions
RUN mkdir -p \
    /data/patroni \
    /var/log/patroni \
    /etc/patroni \
    /var/run/postgresql \
    /opt/secretpg \
    /pgbackrest/repo \
    /pgbackrest/log && \
    chown -R postgres:postgres \
    /data \
    /var/log/patroni \
    /etc/patroni \
    /var/run/postgresql \
    /opt/secretpg \
    /pgbackrest && \
    chmod 700 /data && \
    chmod 700 /data/patroni && \
    chmod 2775 /var/run/postgresql && \
    chmod 700 /opt/secretpg

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set environment variables
ENV PGDATA=/data/patroni
ENV PATH="/usr/lib/postgresql/17/bin:${PATH}"
ENV PG_VERSION=17

# Expose ports
# 5432 - PostgreSQL
# 8008 - Patroni REST API
EXPOSE 5432 8008

# Switch to postgres user
USER postgres

# Entrypoint
ENTRYPOINT ["/entrypoint.sh"]
