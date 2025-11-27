# FRONTEND INFORMATION ARCHITECTURE
# Dashboard UI Requirements cho IaaS Platform

## 📊 OVERVIEW DASHBOARD (Trang chính)

### 1. **System Summary Cards**
```
┌─────────────────────────────────────────────────────────────────┐
│  📦 Total Clusters: 12    |  🔴 Running: 10  |  ⚠️ Issues: 2    │
│  💾 Total Storage: 1.2TB  |  💰 Est. Cost: $450/month          │
│  📈 Uptime: 99.8%        |  ⚡ Avg Response: 45ms              │
└─────────────────────────────────────────────────────────────────┘
```

**Hiển thị:**
- Tổng số clusters (PostgreSQL, Nginx, Docker Services)
- Tổng storage đã sử dụng
- Chi phí ước tính (tính theo resource usage)
- System health score
- Average response time

### 2. **Resource Usage Chart**
```
CPU Usage:     ████████░░ 80%
Memory Usage:  ██████░░░░ 60%
Disk Usage:    ████░░░░░░ 40%
Network I/O:   ███░░░░░░░ 30%
```

**Hiển thị:**
- Real-time resource usage với charts
- Historical data (24h, 7d, 30d)
- Alert thresholds với visual indicators

---

## 🗄️ POSTGRESQL CLUSTER DASHBOARD

### 3. **Cluster Cards - Enhanced View**
```
┌───────────────────────────────────────────────────────────┐
│ 🐘 prod-database-cluster                                  │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Status: 🟢 Running | Version: PostgreSQL 17              │
│                                                           │
│ TOPOLOGY:                                                 │
│   Primary:   patroni-1 (172.18.0.10:5432) ✓ Healthy     │
│   Replicas:  patroni-2 (172.18.0.11:5432) ✓ Streaming   │
│              patroni-3 (172.18.0.12:5432) ✓ Streaming   │
│                                                           │
│ REPLICATION STATUS:                                       │
│   Mode: Async | Lag: <100KB | Delay: <1s                │
│                                                           │
│ PERFORMANCE:                                              │
│   Connections: 45/100 | QPS: 1,250 | Latency: 12ms      │
│   Cache Hit Rate: 98.5% | Active Transactions: 8         │
│                                                           │
│ STORAGE:                                                  │
│   Total Size: 45GB | Growth: +2GB/week | Backup: 3h ago │
│   Databases: 5 | Tables: 128 | Indexes: 256             │
│                                                           │
│ ENDPOINTS:                                                │
│   Write:  haproxy:5000 (Primary)                        │
│   Read:   haproxy:5001 (Load Balanced)                  │
│                                                           │
│ [Details] [Logs] [Metrics] [Backup] [Scale] [Settings]  │
└───────────────────────────────────────────────────────────┘
```

**Thông tin cần hiển thị:**

#### A. **Basic Info**
- Cluster name
- Status badge (Running, Stopped, Error, Degraded)
- PostgreSQL version
- Created/Updated time
- Uptime duration

#### B. **Topology & Health**
- Node list với roles (Primary/Replica)
- Health status mỗi node (✓ Healthy, ⚠️ Warning, ✗ Down)
- Replication status (Streaming, Catching up, Disconnected)
- Network endpoints (IP:Port)

#### C. **Replication Metrics**
- Replication mode (Async/Sync)
- Replication lag (bytes và seconds)
- Sync state của mỗi replica
- WAL position/timeline

#### D. **Performance Metrics**
- Active connections / Max connections
- Queries per second (QPS)
- Average query latency
- Cache hit ratio
- Active transactions
- Database locks

#### E. **Storage Info**
- Total database size
- Growth rate (per day/week)
- Number of databases
- Number of tables
- Last backup time & status
- Backup retention policy

#### F. **Resource Usage**
Per-node metrics:
- CPU usage (%)
- Memory usage (%)
- Disk I/O (read/write MB/s)
- Network traffic

---

## 🌐 NGINX INSTANCES DASHBOARD

### 4. **Nginx Cards - Enhanced View**
```
┌───────────────────────────────────────────────────────────┐
│ 🔷 api-gateway-nginx                                      │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Status: 🟢 Running | Port: 80, 443                       │
│                                                           │
│ DOMAINS (3):                                              │
│   ✓ api.example.com (SSL: Valid until 2026-03-15)       │
│   ✓ www.example.com (SSL: Valid until 2026-03-15)       │
│   ✓ admin.example.com (No SSL)                          │
│                                                           │
│ ROUTES (5):                                               │
│   /api       → backend-service (2 servers, RR)           │
│   /admin     → admin-service (1 server)                  │
│   /static    → cdn-service (3 servers, LC)               │
│   /websocket → ws-service (sticky sessions)              │
│   /health    → health-check                              │
│                                                           │
│ UPSTREAMS:                                                │
│   backend-service:                                        │
│     ✓ backend1:3000 (weight: 2) - Healthy               │
│     ✓ backend2:3000 (weight: 1) - Healthy               │
│     ✗ backend3:3000 (weight: 1) - Down (5m ago)         │
│   Algorithm: Round Robin | Health Check: 30s            │
│                                                           │
│ TRAFFIC STATS (24h):                                      │
│   Requests: 1.2M | Success: 98.5% | Errors: 1.5%        │
│   Avg Response: 45ms | P95: 120ms | P99: 350ms          │
│   Bandwidth: ↓ 120GB ↑ 45GB                             │
│                                                           │
│ SECURITY:                                                 │
│   CORS: Enabled | Rate Limit: 100 req/min               │
│   IP Whitelist: 3 IPs | DDoS Protection: Active         │
│                                                           │
│ [Config] [Routes] [SSL] [Logs] [Metrics] [Restart]      │
└───────────────────────────────────────────────────────────┘
```

**Thông tin cần hiển thị:**

#### A. **Basic Info**
- Instance name
- Status (Running, Stopped, Reloading)
- Listening ports (80, 443, custom)
- Nginx version
- Uptime

#### B. **Domains**
- List domains
- SSL certificate status (Valid, Expiring soon, Invalid)
- SSL expiry dates
- Auto-renewal status

#### C. **Routes Configuration**
- Path patterns
- Target upstream services
- HTTP methods allowed
- Rate limiting per route
- Caching rules

#### D. **Upstreams Health**
- Backend server list
- Health status mỗi server (Up/Down)
- Load balancing algorithm (RR, LC, IP Hash, Sticky)
- Server weights
- Health check config (interval, timeout)
- Last health check time

#### E. **Traffic Metrics**
- Total requests (24h, 7d, 30d)
- Success rate (2xx, 3xx)
- Error rate (4xx, 5xx)
- Average response time
- Percentile latencies (P50, P95, P99)
- Bandwidth usage (In/Out)
- Top request paths

#### F. **Security Settings**
- CORS enabled/disabled
- Allowed origins
- Rate limiting rules
- IP whitelist/blacklist
- DDoS protection status
- WAF rules active

---

## 🐋 DOCKER SERVICES DASHBOARD

### 5. **Service Cards - Enhanced View**
```
┌───────────────────────────────────────────────────────────┐
│ 🐳 app-redis-cache                                        │
│ ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ │
│ Status: 🟢 Running | Image: redis:7-alpine               │
│                                                           │
│ CONTAINER INFO:                                           │
│   ID: a1b2c3d4e5f6 (short)                               │
│   Created: 2024-11-20 10:30:45                           │
│   Uptime: 4d 13h 25m                                     │
│   Restart Policy: always | Restart Count: 0             │
│                                                           │
│ NETWORK:                                                  │
│   Ports: 6379 → 6380 (mapped)                           │
│   Network: iaas-network                                  │
│   IP Address: 172.18.0.25                                │
│   Connected Services: 3                                   │
│                                                           │
│ RESOURCES:                                                │
│   CPU: ████░░░░░░ 40% (0.4/1.0 cores)                   │
│   Memory: ██████░░░░ 60% (307/512 MB)                   │
│   Disk I/O: Read 45MB/s | Write 12MB/s                  │
│   Network I/O: ↓ 2.5MB/s ↑ 800KB/s                      │
│                                                           │
│ ENVIRONMENT:                                              │
│   REDIS_PASSWORD: ******** (hidden)                      │
│   REDIS_MAXMEMORY: 256mb                                 │
│   REDIS_MAXMEMORY_POLICY: allkeys-lru                    │
│   (3 more variables)                                      │
│                                                           │
│ VOLUMES:                                                  │
│   redis-data → /data (10GB used / 50GB)                 │
│                                                           │
│ HEALTH:                                                   │
│   Last Check: 2s ago | Status: Healthy                   │
│   Check Command: redis-cli ping                          │
│   Check Interval: 30s | Timeout: 5s                      │
│                                                           │
│ [Logs] [Exec] [Env] [Restart] [Stop] [Remove]           │
└───────────────────────────────────────────────────────────┘
```

**Thông tin cần hiển thị:**

#### A. **Container Info**
- Service name
- Container ID (short & full)
- Image name & tag
- Status (Running, Stopped, Restarting, Dead)
- Created time
- Uptime duration
- Restart policy & count

#### B. **Network Configuration**
- Port mappings (container:host)
- Network name
- IP address
- DNS names
- Connected containers/services

#### C. **Resource Usage**
Real-time metrics:
- CPU usage (cores & percentage)
- Memory usage (used/limit)
- Disk I/O (read/write rates)
- Network I/O (in/out bandwidth)
- PIDs count

#### D. **Environment Variables**
- List all env vars
- Mask sensitive values (passwords, tokens)
- Show/hide toggle

#### E. **Volumes & Storage**
- Volume names & mount paths
- Storage usage per volume
- Volume type (named, bind mount)
- Read/write permissions

#### F. **Health Checks**
- Health status (Healthy, Unhealthy, Starting)
- Last check time
- Health check command
- Check interval & timeout
- Failure threshold

#### G. **Logs Preview**
- Last 50-100 lines
- Real-time streaming option
- Search/filter logs
- Download logs

---

## 📈 MONITORING & METRICS VIEWS

### 6. **Cluster Health Dashboard**
```
┌─────────────────────────────────────────────────────────┐
│  CLUSTER HEALTH OVERVIEW                                 │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                          │
│  System Status:  🟢 Healthy (98.5% uptime)              │
│  Active Alerts:  ⚠️ 3 warnings, 0 critical             │
│                                                          │
│  RESOURCE TRENDS (7 days):                              │
│  ┌──────────────────────────────────────────┐          │
│  │ CPU    ▁▂▃▄▅▆▇█▇▆▅▄▃▂▁                   │          │
│  │ Memory ▃▃▄▅▆▆▇▇▇▇▆▆▅▄▃                   │          │
│  │ Disk   ▁▁▂▂▂▃▃▃▄▄▄▅▅▅▆                   │          │
│  └──────────────────────────────────────────┘          │
│                                                          │
│  TOP PERFORMANCE ISSUES:                                 │
│  1. ⚠️ High replication lag on prod-db-replica-2       │
│  2. ⚠️ Memory usage >80% on api-gateway-nginx          │
│  3. ⚠️ Disk space <20% on mongo-service                │
│                                                          │
│  BACKUP STATUS:                                          │
│  ✓ Last backup: 3h ago (Success)                       │
│  ✓ Next backup: in 21h                                 │
│  ✓ Backup retention: 7 days / 12 backups               │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

### 7. **Performance Metrics Charts**
- **Time-series graphs** cho CPU, Memory, Disk, Network
- **Heatmaps** cho query performance
- **Percentile charts** cho latency (P50, P95, P99)
- **Connection pool usage** over time
- **Replication lag trends**

---

## 🔔 ALERTS & NOTIFICATIONS

### 8. **Alert Center**
```
┌─────────────────────────────────────────────────────────┐
│  ACTIVE ALERTS                                           │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                          │
│  🔴 CRITICAL (0)                                         │
│                                                          │
│  ⚠️ WARNING (3)                                          │
│  • High replication lag detected                        │
│    Cluster: prod-db | Lag: 500KB | Since: 5m ago       │
│    [View Details] [Acknowledge]                         │
│                                                          │
│  • Memory usage threshold exceeded                       │
│    Service: api-gateway | Usage: 85% | Since: 15m ago  │
│    [View Metrics] [Scale Up]                            │
│                                                          │
│  • SSL certificate expiring soon                         │
│    Domain: api.example.com | Expires: in 14 days       │
│    [Renew Certificate]                                  │
│                                                          │
│  🟡 INFO (5)                                             │
│  • Backup completed successfully                         │
│  • Cluster scaled from 3 to 5 nodes                     │
│  • Configuration updated                                 │
│  (2 more...)                                            │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

**Alert Types:**
- 🔴 Critical: Service down, data loss risk
- ⚠️ Warning: Performance degradation, approaching limits
- 🟡 Info: Successful operations, config changes

---

## 🛠️ OPERATIONS & ACTIONS

### 9. **Quick Actions Panel**
```
┌─────────────────────────────────────────────────────────┐
│  QUICK ACTIONS                                           │
│  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │
│                                                          │
│  🐘 PostgreSQL                                           │
│  [+ New Cluster] [Scale Cluster] [Manual Failover]     │
│  [Run Backup] [Restore Point-in-Time]                  │
│                                                          │
│  🌐 Nginx                                                │
│  [+ New Instance] [Add Domain] [Update SSL]            │
│  [Edit Routes] [Security Policy]                        │
│                                                          │
│  🐋 Docker Services                                      │
│  [+ New Service] [Bulk Start/Stop] [Update Images]     │
│  [Network Config] [Volume Management]                   │
│                                                          │
│  📊 System                                               │
│  [View Logs] [Export Metrics] [System Settings]        │
│  [User Management] [API Keys]                           │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

---

## 📋 DETAILED VIEWS - Modal/Pages

### 10. **Cluster Details Page**
Khi click vào cluster card, hiển thị full-page view:

**Tabs:**
1. **Overview**: Summary info như trên
2. **Nodes**: 
   - Detailed node list với status
   - Resource usage per node
   - Patroni config
   - Health check results
3. **Replication**:
   - Topology diagram
   - Replication lag graphs
   - WAL streaming status
   - Sync vs Async status
4. **Performance**:
   - Query statistics
   - Slow query log
   - Connection pool stats
   - Cache hit rate trends
5. **Databases**:
   - Database list với size
   - Table statistics
   - Index usage
   - Vacuum/Analyze status
6. **Users & Permissions**:
   - User list
   - Role assignments
   - Permission matrix
7. **Backups**:
   - Backup history
   - Recovery points
   - Restore options
   - Backup schedules
8. **Logs**:
   - PostgreSQL logs
   - Patroni logs
   - Container logs
   - Search & filter
9. **Settings**:
   - Configuration parameters
   - Replication settings
   - Resource limits
   - Maintenance windows

### 11. **Node Status Indicators**
Visual indicators cho từng component:

```
Status Colors:
🟢 Green:  Healthy, Running normally
🟡 Yellow: Warning, Degraded performance
🔴 Red:    Critical, Down/Failed
⚪ Gray:   Stopped, Disabled
🔵 Blue:   Starting, Initializing
```

---

## 💡 KEY INFORMATION PRIORITIES

### **Critical Info (Always Visible)**
1. Service status (Up/Down)
2. Health status (Healthy/Unhealthy)
3. Active alerts count
4. Primary/Replica roles
5. Connection endpoints

### **Important Info (Card Level)**
1. Resource usage (CPU, Memory)
2. Performance metrics (Latency, QPS)
3. Replication status
4. Storage usage
5. Recent operations

### **Detailed Info (Modal/Page Level)**
1. Full configuration
2. Historical metrics
3. Complete logs
4. Audit trail
5. Cost breakdown

---

## 🎨 UI/UX ENHANCEMENTS

### 12. **Interactive Features**
- **Drag-and-drop** topology diagram
- **Real-time updates** via WebSocket
- **Search & filter** across all resources
- **Bulk operations** (start/stop multiple services)
- **Favorites/bookmarks** for frequently accessed resources
- **Custom dashboards** with widgets
- **Dark mode** support

### 13. **Data Visualization**
- **Time-series charts** (Line, Area)
- **Resource gauges** (CPU, Memory dials)
- **Heatmaps** (Query performance)
- **Network topology graphs**
- **Traffic flow diagrams**

---

## 📱 RESPONSIVE DESIGN

### Mobile View Priority:
1. Service status overview
2. Critical alerts
3. Quick actions (Start/Stop/Restart)
4. Resource usage summary
5. Recent logs (last 20 lines)

---

## 🔐 SECURITY & AUDIT

### 14. **Audit Trail View**
```
2024-11-24 10:30:45 | admin | CREATE | PostgreSQL Cluster "prod-db"
2024-11-24 10:32:12 | admin | START  | PostgreSQL Cluster "prod-db"
2024-11-24 10:45:00 | user1  | SCALE  | Cluster "prod-db" from 3 to 5 nodes
2024-11-24 11:00:00 | system | BACKUP | Cluster "prod-db" completed
```

**Fields:**
- Timestamp
- User/System actor
- Action type (CREATE, UPDATE, DELETE, START, STOP, etc.)
- Resource affected
- Result (Success/Failed)
- Details/Changes

---

## 🎯 SUMMARY - CORE DATA NEEDED FROM BACKEND

### PostgreSQL Cluster:
✅ Status, Version, Uptime
✅ Node list (roles, health, endpoints)
✅ Replication status (lag, sync state)
✅ Performance metrics (QPS, latency, connections)
✅ Storage size & growth rate
✅ Backup status & history
✅ Resource usage per node

### Nginx:
✅ Status, Ports
✅ Domain list with SSL status
✅ Routes & upstreams config
✅ Backend health checks
✅ Traffic stats (requests, errors, latency)
✅ Security settings (CORS, rate limit)

### Docker Services:
✅ Container status, uptime
✅ Image & version
✅ Port mappings & network info
✅ Resource usage (CPU, Memory, Disk, Network)
✅ Environment variables
✅ Volume info
✅ Health check status
✅ Logs

### System-wide:
✅ Total resource usage
✅ Active alerts
✅ Backup status
✅ Cost estimates
✅ Audit logs
