package dto

// CreateClusterRequest for creating PostgreSQL cluster with Patroni/Spilo
type CreateClusterRequest struct {
	ClusterName        string   `json:"cluster_name" binding:"required"`
	PostgreSQLVersion  string   `json:"postgres_version" binding:"required"` // 15, 16, etc
	NodeCount          int      `json:"node_count" binding:"required,min=1,max=10"`
	CPUPerNode         int64    `json:"cpu_per_node" binding:"required"`
	MemoryPerNode      int64    `json:"memory_per_node" binding:"required"`
	StoragePerNode     int      `json:"storage_per_node" binding:"required"` // GB
	InstanceNames      []string `json:"instance_names,omitempty"`
	PostgreSQLPassword string   `json:"postgres_password" binding:"required,min=8"`
	ReplicationMode    string   `json:"replication_mode" binding:"required,oneof=async sync"`
	MaxReplicationLag  int64    `json:"max_replication_lag,omitempty"`
	SynchronousStandby []string `json:"synchronous_standby_names,omitempty"`
	DataDirectory      string   `json:"data_directory,omitempty"`
	VolumeType         string   `json:"volume_type,omitempty"`

	// Patroni/DCS configuration
	DCSType      string   `json:"dcs_type,omitempty"`      // etcd, consul (optional, default: etcd)
	DCSEndpoints []string `json:"dcs_endpoints,omitempty"` // DCS server endpoints
	Namespace    string   `json:"namespace,omitempty"`     // Patroni namespace (default: percona_lab)

	// Patroni Bootstrap Settings
	TTL                  int    `json:"ttl,omitempty"`                     // DCS TTL (default: 30s)
	LoopWait             int    `json:"loop_wait,omitempty"`               // Loop wait (default: 10s)
	RetryTimeout         int    `json:"retry_timeout,omitempty"`           // Retry timeout (default: 10s)
	MaximumLagOnFailover int64  `json:"maximum_lag_on_failover,omitempty"` // Max lag for failover (default: 1MB)
	UsePgRewind          bool   `json:"use_pg_rewind,omitempty"`           // Enable pg_rewind (default: true)
	UseSlots             bool   `json:"use_slots,omitempty"`               // Enable replication slots (default: true)
	WatchdogMode         string `json:"watchdog_mode,omitempty"`           // off, automatic, required (default: off)

	// Patroni Tags
	NoFailover    bool `json:"nofailover,omitempty"`    // Prevent node from becoming primary
	NoLoadBalance bool `json:"noloadbalance,omitempty"` // Prevent read traffic routing
	CloneFrom     bool `json:"clonefrom,omitempty"`     // Allow cloning from this node
	NoSync        bool `json:"nosync,omitempty"`        // Prevent sync replication

	// PgBackRest Configuration
	EnableBackup     bool   `json:"enable_backup,omitempty"`      // Enable PgBackRest backup
	BackupRetention  int    `json:"backup_retention,omitempty"`   // Full backup retention days (default: 7)
	BackupSchedule   string `json:"backup_schedule,omitempty"`    // Cron schedule for backups
	BackupProcessMax int    `json:"backup_process_max,omitempty"` // Parallel backup processes (default: 2)

	// HAProxy Configuration
	EnableHAProxy    bool `json:"enable_haproxy,omitempty"`     // Enable HAProxy for load balancing
	HAProxyPort      int  `json:"haproxy_port,omitempty"`       // HAProxy primary port (default: 5000)
	HAProxyReadPort  int  `json:"haproxy_read_port,omitempty"`  // HAProxy replica port (default: 5001)
	HAProxyStatsPort int  `json:"haproxy_stats_port,omitempty"` // HAProxy stats port (default: 7000)

	// PostgreSQL parameters
	Parameters map[string]string `json:"parameters,omitempty"` // max_connections, shared_buffers, etc

	// Users to create (beyond default postgres user)
	Users []ClusterUser `json:"users,omitempty"`

	// Databases to create (beyond default postgres db)
	Databases []ClusterDatabase `json:"databases,omitempty"`

	// Security
	EnableTLS bool `json:"enable_tls,omitempty"`
}

// ClusterUser defines user with roles
type ClusterUser struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles,omitempty"` // SUPERUSER, CREATEDB, CREATEROLE, LOGIN, REPLICATION
}

// ClusterDatabase defines database with owner
type ClusterDatabase struct {
	Name     string `json:"name" binding:"required"`
	Owner    string `json:"owner" binding:"required"`
	Encoding string `json:"encoding,omitempty"` // UTF8, LATIN1, etc (default: UTF8)
}

// ClusterInfoResponse returns cluster details
type ClusterInfoResponse struct {
	ClusterID         string `json:"cluster_id"`
	InfrastructureID  string `json:"infrastructure_id"`
	ClusterName       string `json:"cluster_name"`
	PostgreSQLVersion string `json:"postgres_version"`
	Status            string `json:"status"`
	ReplicationMode   string `json:"replication_mode"` // async or sync

	// Connection endpoints
	WriteEndpoint ConnectionPoint   `json:"write_endpoint"` // Primary for R/W
	ReadEndpoints []ConnectionPoint `json:"read_endpoints"` // Replicas for read-only

	// Cluster nodes
	Nodes []ClusterNodeInfo `json:"nodes"`

	// DCS info
	DCSType      string   `json:"dcs_type,omitempty"`
	DCSEndpoints []string `json:"dcs_endpoints,omitempty"`

	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
	HAProxyPort       int    `json:"haproxy_port"`
	MaxReplicationLag int64  `json:"max_replication_lag"`
}

// ConnectionPoint represents connection endpoint (primary or replica)
type ConnectionPoint struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	NodeID   string `json:"node_id"`
	NodeRole string `json:"node_role"` // primary or replica
}

// ClusterNodeInfo details for each node
type ClusterNodeInfo struct {
	NodeID           string `json:"node_id"`
	NodeName         string `json:"node_name"`
	ContainerID      string `json:"container_id"`
	Role             string `json:"role"` // primary or replica
	Status           string `json:"status"`
	ReplicationDelay int    `json:"replication_delay"` // bytes
	IsHealthy        bool   `json:"is_healthy"`
}

// ScaleClusterRequest for scaling up/down
type ScaleClusterRequest struct {
	NodeCount int `json:"node_count" binding:"required,min=1,max=10"`
}

// AddNodeRequest for adding a new replica node
type AddNodeRequest struct {
	NodeName string `json:"node_name,omitempty"` // Optional custom name
}

// AddNodeResponse returns info about the new node
type AddNodeResponse struct {
	NodeID      string `json:"node_id"`
	NodeName    string `json:"node_name"`
	ContainerID string `json:"container_id"`
	Role        string `json:"role"`
	Port        int    `json:"port"`
	Status      string `json:"status"`
	Message     string `json:"message"`
}

// RemoveNodeRequest for removing a node
type RemoveNodeRequest struct {
	NodeID string `json:"node_id" binding:"required"`
	Force  bool   `json:"force,omitempty"` // Force remove even if node is unhealthy
}

// RemoveNodeResponse returns result of node removal
type RemoveNodeResponse struct {
	NodeID  string `json:"node_id"`
	Removed bool   `json:"removed"`
	Message string `json:"message"`
}

// TriggerFailoverRequest for manual failover
type TriggerFailoverRequest struct {
	NewPrimaryNodeID string `json:"new_primary_node_id" binding:"required"`
}

// ReplicationStatusResponse shows replication health
type ReplicationStatusResponse struct {
	Primary  string          `json:"primary"`
	Replicas []ReplicaStatus `json:"replicas"`
}

// ReplicaStatus for each replica node
type ReplicaStatus struct {
	NodeName   string  `json:"node_name"`
	State      string  `json:"state"`      // streaming, catchup, etc
	SyncState  string  `json:"sync_state"` // async or sync
	LagBytes   int     `json:"lag_bytes"`
	LagSeconds float64 `json:"lag_seconds"`
	IsHealthy  bool    `json:"is_healthy"`
}

// ClusterStatsResponse aggregated stats
type ClusterStatsResponse struct {
	ClusterID        string      `json:"cluster_id"`
	TotalConnections int         `json:"total_connections"`
	TotalDatabases   int         `json:"total_databases"`
	TotalSizeMB      int         `json:"total_size_mb"`
	Nodes            []NodeStats `json:"nodes"`
}

type NodeStats struct {
	NodeName          string `json:"node_name"`
	Role              string `json:"role"`
	CPUPercent        int    `json:"cpu_percent"`
	MemoryPercent     int    `json:"memory_percent"`
	ActiveConnections int    `json:"active_connections"`
}

type ClusterLogsResponse struct {
	ClusterID string    `json:"cluster_id"`
	Logs      []NodeLog `json:"logs"`
}

// NodeLog individual node logs
type NodeLog struct {
	NodeName  string `json:"node_name"`
	Timestamp string `json:"timestamp"`
	Logs      string `json:"logs"`
}

// CreateUserRequest for creating user in cluster
type CreateUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Password string   `json:"password" binding:"required,min=8"`
	Roles    []string `json:"roles,omitempty"` // SUPERUSER, CREATEDB, etc
}

// CreateClusterDatabaseRequest for creating database in cluster
type CreateClusterDatabaseRequest struct {
	Name     string `json:"name" binding:"required"`
	Owner    string `json:"owner" binding:"required"`
	Encoding string `json:"encoding,omitempty"`
}

// UpdateConfigRequest for updating cluster configuration
type UpdateConfigRequest struct {
	ReplicationMode string            `json:"replication_mode,omitempty" binding:"omitempty,oneof=async sync"`
	Parameters      map[string]string `json:"parameters,omitempty"` // max_connections, shared_buffers, etc
}

// UserInfo represents user information
type UserInfo struct {
	Username  string   `json:"username"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at"`
}

type ClusterDatabaseInfo struct {
	Name      string `json:"name"`
	Owner     string `json:"owner"`
	Encoding  string `json:"encoding"`
	Size      string `json:"size"`
	CreatedAt string `json:"created_at"`
}

// FailoverEvent represents a failover event in cluster history
type FailoverEvent struct {
	ID             string `json:"id"`
	ClusterID      string `json:"cluster_id"`
	OldPrimaryID   string `json:"old_primary_id"`
	OldPrimaryName string `json:"old_primary_name"`
	NewPrimaryID   string `json:"new_primary_id"`
	NewPrimaryName string `json:"new_primary_name"`
	Reason         string `json:"reason"`       // manual, automatic, node_failure
	TriggeredBy    string `json:"triggered_by"` // system, user
	OccurredAt     string `json:"occurred_at"`
}

// NodeActionRequest for stop/start node
type NodeActionRequest struct {
	NodeID string `json:"node_id" binding:"required"`
}

// PatroniHealthResponse from Patroni API
type PatroniHealthResponse struct {
	State        string `json:"state"` // running, stopped
	Role         string `json:"role"`  // master, replica
	Timeline     int    `json:"timeline"`
	XlogLocation int64  `json:"xlog_location"`
}

// ExecuteQueryRequest for running SQL on cluster
type ExecuteQueryRequest struct {
	Query    string `json:"query" binding:"required"`
	Database string `json:"database"` // optional, default postgres
	NodeID   string `json:"node_id"`  // optional, specific node to run on
}

// QueryResult represents query execution result
type QueryResult struct {
	Columns  []string        `json:"columns"`
	Rows     [][]interface{} `json:"rows"`
	RowCount int             `json:"row_count"`
	Duration string          `json:"duration"`
	NodeName string          `json:"node_name"`
	NodeRole string          `json:"node_role"`
}

// ReplicationTestResult shows replication status across nodes
type ReplicationTestResult struct {
	PrimaryNode   string           `json:"primary_node"`
	TestTable     string           `json:"test_table"`
	TestData      string           `json:"test_data"`
	NodeResults   []NodeSyncResult `json:"node_results"`
	AllSynced     bool             `json:"all_synced"`
	TestTimestamp string           `json:"test_timestamp"`
}

// NodeSyncResult shows sync status for each node
type NodeSyncResult struct {
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Role      string `json:"role"`
	HasData   bool   `json:"has_data"`
	RowCount  int    `json:"row_count"`
	SyncDelay string `json:"sync_delay"`
}
