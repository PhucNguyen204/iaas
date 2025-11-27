package entities

import (
	"time"
)

type PostgreSQLCluster struct {
	ID                 string         `gorm:"primaryKey;type:varchar(36)"`
	InfrastructureID   string         `gorm:"type:varchar(36);not null;index"`
	Infrastructure     Infrastructure `gorm:"foreignKey:InfrastructureID"`
	ClusterName        string         `gorm:"type:varchar(255)"`
	NodeCount          int            `gorm:"not null"`
	PrimaryNodeID      string         `gorm:"type:varchar(100)"`
	Version            string         `gorm:"type:varchar(20);not null"`
	DatabaseName       string         `gorm:"type:varchar(100);not null"`
	Username           string         `gorm:"type:varchar(100);not null"`
	Password           string         `gorm:"type:varchar(255);not null"`
	ReplicationMode    string         `gorm:"type:varchar(10);default:'async'"` // async or sync
	HAProxyPort        int            `gorm:"not null"`
	NetworkID          string         `gorm:"type:varchar(255)"`
	DCSType            string         `gorm:"type:varchar(20);default:'etcd'"` // etcd, consul
	DCSEndpoints       string         `gorm:"type:varchar(500)"`               // JSON array
	MaxReplicationLag  int64          `gorm:"default:0"`
	SynchronousStandby string         `gorm:"type:varchar(255)"`
	DataDirectory      string         `gorm:"type:varchar(255)"`
	VolumeType         string         `gorm:"type:varchar(50)"`
	StorageSize        int            `gorm:"default:0"`
	CPULimit           int64          `gorm:"default:0"`
	MemoryLimit        int64          `gorm:"default:0"`
	CreatedAt          time.Time      `gorm:"autoCreateTime"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime"`
}

type ClusterNode struct {
	ID               string            `gorm:"primaryKey;type:varchar(36)"`
	ClusterID        string            `gorm:"type:varchar(36);not null;index"`
	Cluster          PostgreSQLCluster `gorm:"foreignKey:ClusterID"`
	ContainerID      string            `gorm:"type:varchar(100)"`
	Role             string            `gorm:"type:varchar(20);not null"`
	Port             int               `gorm:"not null"`
	VolumeID         string            `gorm:"type:varchar(255)"`
	ReplicationDelay int64             `gorm:"default:0"`
	IsHealthy        bool              `gorm:"default:true"`
	CreatedAt        time.Time         `gorm:"autoCreateTime"`
	UpdatedAt        time.Time         `gorm:"autoUpdateTime"`
}

type EtcdNode struct {
	ID          string    `gorm:"primaryKey;type:varchar(36)"`
	ClusterID   string    `gorm:"type:varchar(36);not null;index"`
	ContainerID string    `gorm:"type:varchar(100)"`
	Port        int       `gorm:"not null"`
	VolumeID    string    `gorm:"type:varchar(255)"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// FailoverEvent tracks failover history for a cluster
type FailoverEvent struct {
	ID             string            `gorm:"primaryKey;type:varchar(36)"`
	ClusterID      string            `gorm:"type:varchar(36);not null;index"`
	Cluster        PostgreSQLCluster `gorm:"foreignKey:ClusterID"`
	OldPrimaryID   string            `gorm:"type:varchar(36)"`
	OldPrimaryName string            `gorm:"type:varchar(100)"`
	NewPrimaryID   string            `gorm:"type:varchar(36)"`
	NewPrimaryName string            `gorm:"type:varchar(100)"`
	Reason         string            `gorm:"type:varchar(50)"` // manual, automatic, node_failure
	TriggeredBy    string            `gorm:"type:varchar(50)"` // system, user
	OccurredAt     time.Time         `gorm:"autoCreateTime"`
}
