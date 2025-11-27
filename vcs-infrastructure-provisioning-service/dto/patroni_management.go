package dto

// PatroniManagementRequest for Patroni control operations
type PatroniManagementRequest struct {
	Action      string `json:"action" binding:"required,oneof=switchover reinit pause resume restart reload"`
	NodeID      string `json:"node_id,omitempty"`      // Target node ID for specific operations
	CandidateID string `json:"candidate_id,omitempty"` // Candidate node for switchover
	Force       bool   `json:"force,omitempty"`        // Force operation
	ScheduledAt string `json:"scheduled_at,omitempty"` // ISO8601 timestamp for scheduled switchover
}

// SwitchoverRequest for manual switchover
type SwitchoverRequest struct {
	LeaderNode    string `json:"leader_node,omitempty"`  // Current leader (optional, auto-detected)
	CandidateNode string `json:"candidate_node"`         // New leader candidate
	ScheduledAt   string `json:"scheduled_at,omitempty"` // ISO8601 for scheduled switchover
}

// ReinitRequest for reinitializing a node
type ReinitRequest struct {
	NodeID string `json:"node_id" binding:"required"` // Node to reinitialize
	Force  bool   `json:"force,omitempty"`            // Force reinit even if node is healthy
}

// BackupRequest for PgBackRest operations
type BackupRequest struct {
	Type   string `json:"type" binding:"required,oneof=full incr diff"` // full, incr, diff
	Stanza string `json:"stanza,omitempty"`                             // Stanza name (default: cluster scope)
}

// RestoreRequest for PgBackRest restore
type RestoreRequest struct {
	BackupSet string            `json:"backup_set,omitempty"` // Specific backup set (default: latest)
	Target    string            `json:"target,omitempty"`     // Point-in-time recovery target
	Options   map[string]string `json:"options,omitempty"`    // Additional restore options
}

// PatroniStatusResponse shows Patroni cluster status
type PatroniStatusResponse struct {
	Scope    string          `json:"scope"`
	Members  []PatroniMember `json:"members"`
	SystemID string          `json:"system_id,omitempty"`
	Timeline int             `json:"timeline"`
	Paused   bool            `json:"paused"`
	PausedAt string          `json:"paused_at,omitempty"`
}

// PatroniMember represents a Patroni cluster member
type PatroniMember struct {
	Name           string      `json:"name"`
	Role           string      `json:"role"`  // Leader, Replica, Sync Standby
	State          string      `json:"state"` // running, stopped, starting
	Host           string      `json:"host"`
	Port           int         `json:"port"`
	Timeline       int         `json:"timeline"`
	Lag            int         `json:"lag"` // Replication lag in bytes
	PendingRestart bool        `json:"pending_restart"`
	Tags           PatroniTags `json:"tags,omitempty"`
}

// PatroniTags for node configuration
type PatroniTags struct {
	NoFailover    bool `json:"nofailover"`
	NoLoadBalance bool `json:"noloadbalance"`
	CloneFrom     bool `json:"clonefrom"`
	NoSync        bool `json:"nosync"`
}

type PgBackRestBackupInfoResponse struct {
	Stanza    string                 `json:"stanza"`
	Status    string                 `json:"status"`
	Backups   []PgBackRestBackupInfo `json:"backups"`
	TotalSize string                 `json:"total_size"`
}

type PgBackRestBackupInfo struct {
	Label      string `json:"label"`
	Type       string `json:"type"` 
	Timestamp  string `json:"timestamp"`
	Size       string `json:"size"`
	Repository string `json:"repository"`
}
