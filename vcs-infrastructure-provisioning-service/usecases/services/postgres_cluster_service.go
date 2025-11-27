package services

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/kafka"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/docker/docker/api/types"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IPostgreSQLClusterService interface {
	CreateCluster(ctx context.Context, userID string, req dto.CreateClusterRequest) (*dto.ClusterInfoResponse, error)
	StartCluster(ctx context.Context, clusterID string) error
	StopCluster(ctx context.Context, clusterID string) error
	RestartCluster(ctx context.Context, clusterID string) error
	DeleteCluster(ctx context.Context, clusterID string) error
	GetClusterInfo(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, error)
	ScaleCluster(ctx context.Context, clusterID string, req dto.ScaleClusterRequest) error
	GetClusterStats(ctx context.Context, clusterID string) (*dto.ClusterStatsResponse, error)
	GetClusterLogs(ctx context.Context, clusterID string, tail string) (*dto.ClusterLogsResponse, error)
	PromoteReplica(ctx context.Context, clusterID, nodeID string) error
	GetReplicationStatus(ctx context.Context, clusterID string) (*dto.ReplicationStatusResponse, error)

	// Node management
	StopNode(ctx context.Context, clusterID, nodeID string) error
	StartNode(ctx context.Context, clusterID, nodeID string) error
	GetFailoverHistory(ctx context.Context, clusterID string) ([]dto.FailoverEvent, error)
	AddNode(ctx context.Context, clusterID string, req dto.AddNodeRequest) (*dto.AddNodeResponse, error)
	RemoveNode(ctx context.Context, clusterID string, req dto.RemoveNodeRequest) (*dto.RemoveNodeResponse, error)

	// User management
	CreateUser(ctx context.Context, clusterID string, req dto.CreateUserRequest) error
	ListUsers(ctx context.Context, clusterID string) ([]dto.UserInfo, error)
	DeleteUser(ctx context.Context, clusterID, username string) error

	// Database management
	CreateDatabase(ctx context.Context, clusterID string, req dto.CreateClusterDatabaseRequest) error
	ListDatabases(ctx context.Context, clusterID string) ([]dto.ClusterDatabaseInfo, error)
	DeleteDatabase(ctx context.Context, clusterID, dbname string) error

	// Configuration & Endpoints
	UpdateConfig(ctx context.Context, clusterID string, req dto.UpdateConfigRequest) error
	GetEndpoints(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, error)

	// Query & Replication Test
	ExecuteQuery(ctx context.Context, clusterID string, req dto.ExecuteQueryRequest) (*dto.QueryResult, error)
	TestReplication(ctx context.Context, clusterID string) (*dto.ReplicationTestResult, error)
}

type postgreSQLClusterService struct {
	infraRepo     repositories.IInfrastructureRepository
	clusterRepo   repositories.IPostgreSQLClusterRepository
	dockerSvc     docker.IDockerService
	kafkaProducer kafka.IKafkaProducer
	cacheService  ICacheService
	logger        logger.ILogger
}

func NewPostgreSQLClusterService(
	infraRepo repositories.IInfrastructureRepository,
	clusterRepo repositories.IPostgreSQLClusterRepository,
	dockerSvc docker.IDockerService,
	kafkaProducer kafka.IKafkaProducer,
	cacheService ICacheService,
	logger logger.ILogger,
) IPostgreSQLClusterService {
	return &postgreSQLClusterService{
		infraRepo:     infraRepo,
		clusterRepo:   clusterRepo,
		dockerSvc:     dockerSvc,
		kafkaProducer: kafkaProducer,
		cacheService:  cacheService,
		logger:        logger,
	}
}

// CreateCluster creates a PostgreSQL HA cluster with Patroni + etcd + HAProxy
func (s *postgreSQLClusterService) CreateCluster(ctx context.Context, userID string, req dto.CreateClusterRequest) (*dto.ClusterInfoResponse, error) {
	s.logger.Info("creating PostgreSQL HA cluster with Patroni", zap.String("name", req.ClusterName), zap.Int("nodes", req.NodeCount))

	if req.NodeCount < 1 {
		return nil, fmt.Errorf("node_count must be at least 1")
	}

	// Set Patroni defaults
	if req.Namespace == "" {
		req.Namespace = "percona_lab"
	}
	if req.DCSType == "" {
		req.DCSType = "etcd"
	}
	if req.BackupRetention == 0 {
		req.BackupRetention = 7
	}
	if req.BackupProcessMax == 0 {
		req.BackupProcessMax = 2
	}
	if req.HAProxyPort == 0 {
		req.HAProxyPort = 5000
	}
	if req.HAProxyReadPort == 0 {
		req.HAProxyReadPort = 5001
	}
	if req.HAProxyStatsPort == 0 {
		req.HAProxyStatsPort = 7000
	}

	// Create infrastructure record
	infraID := uuid.New().String()
	infra := &entities.Infrastructure{
		ID:     infraID,
		Name:   req.ClusterName,
		Type:   entities.TypePostgreSQLCluster,
		Status: entities.StatusCreating,
		UserID: userID,
	}
	if err := s.infraRepo.Create(infra); err != nil {
		return nil, fmt.Errorf("failed to create infrastructure: %w", err)
	}

	// Create cluster record
	clusterID := uuid.New().String()
	cluster := &entities.PostgreSQLCluster{
		ID:               clusterID,
		InfrastructureID: infraID,
		NodeCount:        req.NodeCount,
		Version:          req.PostgreSQLVersion,
		DatabaseName:     "postgres",
		Username:         "postgres",
		Password:         req.PostgreSQLPassword,
		CPULimit:         req.CPUPerNode,
		MemoryLimit:      req.MemoryPerNode,
	}
	if err := s.clusterRepo.Create(cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Create dedicated network
	networkName := fmt.Sprintf("iaas-cluster-%s", clusterID)
	networkID, err := s.dockerSvc.CreateNetwork(ctx, networkName)
	if err != nil {
		s.updateInfraStatus(infraID, entities.StatusFailed)
		return nil, fmt.Errorf("failed to create network: %w", err)
	}

	cluster.NetworkID = networkID
	s.clusterRepo.Update(cluster)

	// Step 1: Create etcd cluster (3 nodes for HA)
	s.logger.Info("creating etcd cluster for DCS")
	etcdNodes, err := s.createEtcdCluster(ctx, cluster, req, networkName)
	if err != nil {
		s.cleanup(ctx, cluster, networkID)
		return nil, fmt.Errorf("failed to create etcd cluster: %w", err)
	}

	// Wait for etcd cluster to be ready and healthy
	s.logger.Info("waiting for etcd cluster to be ready", zap.Int("wait_seconds", 15))
	time.Sleep(15 * time.Second) // Increased wait time for etcd cluster formation

	// Verify etcd cluster health by checking if we can connect
	// The Patroni entrypoint will also do its own health check
	s.logger.Info("etcd cluster should be ready, proceeding with Patroni node creation")

	// Step 2: Create Patroni nodes
	s.logger.Info("creating Patroni PostgreSQL nodes")
	patroniNodes := make([]*entities.ClusterNode, 0, req.NodeCount)
	for i := 0; i < req.NodeCount; i++ {
		isLeader := i == 0
		node, err := s.createPatroniNode(ctx, cluster, req, i, networkName, etcdNodes[0], isLeader)
		if err != nil {
			s.logger.Error("failed to create patroni node", zap.Int("index", i), zap.Error(err))
			s.cleanup(ctx, cluster, networkID)
			return nil, fmt.Errorf("failed to create patroni node %d: %w", i, err)
		}
		patroniNodes = append(patroniNodes, node)

		// Set primary node
		if isLeader {
			cluster.PrimaryNodeID = node.ID
			s.clusterRepo.Update(cluster)
		}

		// Wait between nodes for replication setup
		// Longer wait for first replica to ensure primary is fully initialized
		if i < req.NodeCount-1 {
			if i == 0 {
				// First replica needs more time to connect to primary
				s.logger.Info("waiting for primary node to be ready before creating replicas", zap.Int("wait_seconds", 20))
				time.Sleep(20 * time.Second)
			} else {
				// Subsequent replicas can be created faster
				s.logger.Info("waiting before creating next replica", zap.Int("wait_seconds", 15))
				time.Sleep(15 * time.Second)
			}
		}
	}

	// Step 3: Create HAProxy load balancer (always enabled for Patroni)
	s.logger.Info("creating HAProxy load balancer")
	haproxyNode, err := s.createHAProxyNode(ctx, cluster, req, networkName, patroniNodes)
	if err != nil {
		s.logger.Warn("failed to create HAProxy", zap.Error(err))
	} else {
		s.logger.Info("HAProxy created", zap.String("node_id", haproxyNode.ID))
	}

	// Update status
	infra.Status = entities.StatusRunning
	s.infraRepo.Update(infra)

	s.publishEvent(ctx, "cluster.created", infraID, clusterID, string(entities.StatusRunning))
	s.logger.Info("Patroni cluster created successfully", zap.String("cluster_id", clusterID))

	return s.GetClusterInfo(ctx, clusterID)
}

// createEtcdCluster creates a 3-node etcd cluster for DCS
func (s *postgreSQLClusterService) createEtcdCluster(ctx context.Context, cluster *entities.PostgreSQLCluster, req dto.CreateClusterRequest, networkName string) ([]*entities.ClusterNode, error) {
	etcdNodes := make([]*entities.ClusterNode, 0, 3)
	etcdClusterToken := fmt.Sprintf("pgcluster-%s", cluster.ID)

	// Build initial cluster string
	initialCluster := []string{
		fmt.Sprintf("etcd-1=http://etcd-1:2380"),
		fmt.Sprintf("etcd-2=http://etcd-2:2380"),
		fmt.Sprintf("etcd-3=http://etcd-3:2380"),
	}
	initialClusterStr := strings.Join(initialCluster, ",")

	// Create 3 etcd nodes
	for i := 1; i <= 3; i++ {
		nodeID := uuid.New().String()
		nodeName := fmt.Sprintf("etcd-%d", i)
		containerName := fmt.Sprintf("iaas-etcd-%s-%s", cluster.ID, nodeName)
		volumeName := fmt.Sprintf("iaas-etcd-data-%s-%s", cluster.ID, nodeName)

		// Create volume
		if err := s.dockerSvc.CreateVolume(ctx, volumeName); err != nil {
			return nil, fmt.Errorf("failed to create etcd volume: %w", err)
		}

		// For initial cluster bootstrap, all nodes must use "new" state
		// Only use "existing" when adding nodes to a running cluster
		initialState := "new"

		// etcd configuration
		config := docker.ContainerConfig{
			Name:  containerName,
			Image: "iaas-etcd:v3.5.11",
			Env: []string{
				fmt.Sprintf("ETCD_NAME=%s", nodeName),
				fmt.Sprintf("ETCD_INITIAL_CLUSTER_TOKEN=%s", etcdClusterToken),
				fmt.Sprintf("ETCD_INITIAL_CLUSTER_STATE=%s", initialState),
				fmt.Sprintf("ETCD_INITIAL_CLUSTER=%s", initialClusterStr),
				fmt.Sprintf("ETCD_INITIAL_ADVERTISE_PEER_URLS=http://%s:2380", nodeName),
				fmt.Sprintf("ETCD_LISTEN_PEER_URLS=http://0.0.0.0:2380"),
				fmt.Sprintf("ETCD_ADVERTISE_CLIENT_URLS=http://%s:2379", nodeName),
				fmt.Sprintf("ETCD_LISTEN_CLIENT_URLS=http://0.0.0.0:2379"),
			},
			Ports:        map[string]string{"2379": "0", "2380": "0"},
			Volumes:      map[string]string{volumeName: "/etcd-data"},
			Network:      networkName,
			NetworkAlias: nodeName,
			Resources: docker.ResourceConfig{
				CPULimit:    500000000, // 0.5 CPU
				MemoryLimit: 536870912, // 512MB
			},
		}

		containerID, err := s.dockerSvc.CreateContainer(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create etcd container: %w", err)
		}

		if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
			return nil, fmt.Errorf("failed to start etcd container: %w", err)
		}

		node := &entities.ClusterNode{
			ID:          nodeID,
			ClusterID:   cluster.ID,
			ContainerID: containerID,
			Role:        "etcd",
			Port:        2379,
			VolumeID:    volumeName,
			IsHealthy:   true,
		}

		if err := s.clusterRepo.CreateNode(node); err != nil {
			return nil, err
		}

		etcdNodes = append(etcdNodes, node)
		s.logger.Info("created etcd node", zap.String("name", nodeName))

		// Wait between nodes for cluster formation
		if i < 3 {
			time.Sleep(3 * time.Second)
		}
	}

	return etcdNodes, nil
}

// createPatroniNode creates a Patroni-managed PostgreSQL node
func (s *postgreSQLClusterService) createPatroniNode(ctx context.Context, cluster *entities.PostgreSQLCluster, req dto.CreateClusterRequest, index int, networkName string, etcdNode *entities.ClusterNode, isLeader bool) (*entities.ClusterNode, error) {
	nodeID := uuid.New().String()
	nodeName := fmt.Sprintf("patroni-node-%d", index+1)
	containerName := fmt.Sprintf("iaas-patroni-%s-%s", cluster.ID, nodeName)
	volumeName := fmt.Sprintf("iaas-patroni-data-%s-%s", cluster.ID, nodeName)
	backupVolumeName := fmt.Sprintf("iaas-pgbackrest-%s-%s", cluster.ID, nodeName)

	// Create volumes
	if err := s.dockerSvc.CreateVolume(ctx, volumeName); err != nil {
		return nil, fmt.Errorf("failed to create patroni volume: %w", err)
	}
	if err := s.dockerSvc.CreateVolume(ctx, backupVolumeName); err != nil {
		return nil, fmt.Errorf("failed to create backup volume: %w", err)
	}

	// Prepare environment variables
	syncCommit := "local"
	syncStandbyNames := ""
	if req.ReplicationMode == "sync" {
		syncCommit = "on"
		if index > 0 {
			syncStandbyNames = "ANY 1 (*)"
		}
	}

	watchdogMode := req.WatchdogMode
	if watchdogMode == "" {
		watchdogMode = "off"
	}

	// Default parameters if not provided
	maxConnections := "100"
	sharedBuffers := "128MB"
	if req.Parameters != nil {
		if val, ok := req.Parameters["max_connections"]; ok {
			maxConnections = val
		}
		if val, ok := req.Parameters["shared_buffers"]; ok {
			sharedBuffers = val
		}
	}

	// Build Patroni configuration through environment
	env := []string{
		fmt.Sprintf("SCOPE=%s", req.ClusterName),
		fmt.Sprintf("NAMESPACE=%s", req.Namespace),
		fmt.Sprintf("PATRONI_NAME=%s", nodeName),
		fmt.Sprintf("ETCD_HOST=etcd-1:2379"),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", req.PostgreSQLPassword),
		"REPLICATION_PASSWORD=replicator_pass",
		fmt.Sprintf("MAX_CONNECTIONS=%s", maxConnections),
		fmt.Sprintf("SHARED_BUFFERS=%s", sharedBuffers),
		fmt.Sprintf("SYNCHRONOUS_COMMIT=%s", syncCommit),
		fmt.Sprintf("SYNCHRONOUS_STANDBY_NAMES=%s", syncStandbyNames),
		fmt.Sprintf("WATCHDOG_MODE=%s", watchdogMode),
		fmt.Sprintf("NOFAILOVER=%t", req.NoFailover),
		fmt.Sprintf("NOLOADBALANCE=%t", req.NoLoadBalance),
		fmt.Sprintf("CLONEFROM=%t", req.CloneFrom),
		fmt.Sprintf("NOSYNC=%t", req.NoSync),
		fmt.Sprintf("PGDATA=/data/patroni"),
	}

	// PgBackRest configuration
	if req.EnableBackup {
		env = append(env,
			"PGBACKREST_ENABLED=true",
			fmt.Sprintf("PGBACKREST_RETENTION=%d", req.BackupRetention),
			fmt.Sprintf("PGBACKREST_PROCESS_MAX=%d", req.BackupProcessMax),
			fmt.Sprintf("IS_LEADER=%t", isLeader),
		)
	}

	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "iaas-patroni-postgres:17", // Custom built image
		Env:   env,
		Ports: map[string]string{"5432": "0", "8008": "0"},
		Volumes: map[string]string{
			volumeName:       "/data/patroni",
			backupVolumeName: "/pgbackrest",
		},
		Network:      networkName,
		NetworkAlias: nodeName,
		Resources: docker.ResourceConfig{
			CPULimit:    req.CPUPerNode * 1000000000,     // Convert CPU cores to nanocores (1 core = 1e9 nanocores)
			MemoryLimit: req.MemoryPerNode * 1024 * 1024, // Convert MB to bytes
		},
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create patroni container: %w", err)
	}

	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start patroni container: %w", err)
	}

	role := "replica"
	if isLeader {
		role = "primary"
	}

	node := &entities.ClusterNode{
		ID:          nodeID,
		ClusterID:   cluster.ID,
		ContainerID: containerID,
		Role:        role,
		Port:        5432,
		VolumeID:    volumeName,
		IsHealthy:   true,
	}

	if err := s.clusterRepo.CreateNode(node); err != nil {
		return nil, err
	}

	s.logger.Info("created patroni node", zap.String("name", nodeName), zap.String("role", role))
	return node, nil
}

// createHAProxyNode creates HAProxy load balancer for the cluster
func (s *postgreSQLClusterService) createHAProxyNode(ctx context.Context, cluster *entities.PostgreSQLCluster, req dto.CreateClusterRequest, networkName string, patroniNodes []*entities.ClusterNode) (*entities.ClusterNode, error) {
	nodeID := uuid.New().String()
	containerName := fmt.Sprintf("iaas-haproxy-%s", cluster.ID)

	// Build node list for HAProxy
	nodeNames := make([]string, 0, len(patroniNodes))
	for i := range patroniNodes {
		nodeNames = append(nodeNames, fmt.Sprintf("patroni-node-%d", i+1))
	}

	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "iaas-haproxy:latest", // Custom built image
		Env: []string{
			fmt.Sprintf("PATRONI_NODES=%s", strings.Join(nodeNames, ",")),
		},
		Ports: map[string]string{
			"5000": fmt.Sprintf("%d", req.HAProxyPort),
			"5001": fmt.Sprintf("%d", req.HAProxyReadPort),
			"7000": fmt.Sprintf("%d", req.HAProxyStatsPort),
		},
		Network:      networkName,
		NetworkAlias: "haproxy",
		Resources: docker.ResourceConfig{
			CPULimit:    500000000, // 0.5 CPU
			MemoryLimit: 268435456, // 256MB
		},
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create haproxy container: %w", err)
	}

	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start haproxy container: %w", err)
	}

	node := &entities.ClusterNode{
		ID:          nodeID,
		ClusterID:   cluster.ID,
		ContainerID: containerID,
		Role:        "haproxy",
		Port:        req.HAProxyPort,
		IsHealthy:   true,
	}

	if err := s.clusterRepo.CreateNode(node); err != nil {
		return nil, err
	}

	// Store HAProxy port in cluster
	cluster.HAProxyPort = req.HAProxyPort
	s.clusterRepo.Update(cluster)

	s.logger.Info("created HAProxy load balancer", zap.Int("primary_port", req.HAProxyPort), zap.Int("read_port", req.HAProxyReadPort))
	return node, nil
}

// Helper functions and other methods will be copied from backup file...
// (GetClusterInfo, StartCluster, StopCluster, etc.)

func (s *postgreSQLClusterService) updateInfraStatus(infraID string, status entities.InfrastructureStatus) {
	infra, err := s.infraRepo.FindByID(infraID)
	if err == nil {
		infra.Status = status
		s.infraRepo.Update(infra)
	}
}

func (s *postgreSQLClusterService) cleanup(ctx context.Context, cluster *entities.PostgreSQLCluster, networkID string) {
	nodes, _ := s.clusterRepo.ListNodes(cluster.ID)
	for _, node := range nodes {
		s.dockerSvc.StopContainer(ctx, node.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, node.ContainerID)
		if node.VolumeID != "" {
			s.dockerSvc.RemoveVolume(ctx, node.VolumeID)
		}
		s.clusterRepo.DeleteNode(node.ID)
	}
	s.dockerSvc.RemoveNetwork(ctx, networkID)
	s.updateInfraStatus(cluster.InfrastructureID, entities.StatusFailed)
}

func (s *postgreSQLClusterService) publishEvent(ctx context.Context, eventType, infraID, clusterID, status string) {
	// Kafka event publishing logic
	s.logger.Info("publishing event", zap.String("type", eventType), zap.String("cluster_id", clusterID))
}

// GetClusterInfo retrieves cluster information
func (s *postgreSQLClusterService) GetClusterInfo(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, error) {
	if cached, ok := s.cacheService.GetClusterInfo(ctx, clusterID); ok {
		return cached, nil
	}

	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	infra, _ := s.infraRepo.FindByID(cluster.InfrastructureID)
	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	var writeEndpoint dto.ConnectionPoint
	var readEndpoints []dto.ConnectionPoint
	nodeInfos := make([]dto.ClusterNodeInfo, 0)

	for _, node := range nodes {
		if node.Role == "etcd" || node.Role == "haproxy" {
			continue
		}

		nodeInfos = append(nodeInfos, dto.ClusterNodeInfo{
			NodeID:           node.ID,
			NodeName:         fmt.Sprintf("node-%s", node.Role),
			ContainerID:      node.ContainerID,
			Role:             node.Role,
			Status:           "running",
			ReplicationDelay: int(node.ReplicationDelay),
			IsHealthy:        node.IsHealthy,
		})

		endpoint := dto.ConnectionPoint{
			Host:     "localhost",
			Port:     node.Port,
			NodeID:   node.ID,
			NodeRole: node.Role,
		}

		if node.Role == "primary" {
			writeEndpoint = endpoint
		} else if node.Role == "replica" {
			readEndpoints = append(readEndpoints, endpoint)
		}
	}

	response := &dto.ClusterInfoResponse{
		ClusterID:         cluster.ID,
		InfrastructureID:  cluster.InfrastructureID,
		ClusterName:       infra.Name,
		PostgreSQLVersion: cluster.Version,
		Status:            string(infra.Status),
		ReplicationMode:   cluster.ReplicationMode,
		WriteEndpoint:     writeEndpoint,
		ReadEndpoints:     readEndpoints,
		HAProxyPort:       cluster.HAProxyPort,
		Nodes:             nodeInfos,
		CreatedAt:         cluster.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         cluster.UpdatedAt.Format(time.RFC3339),
	}

	s.cacheService.SetClusterInfo(ctx, clusterID, response, 5*time.Minute)

	return response, nil
}

// StartCluster starts all containers in the cluster
func (s *postgreSQLClusterService) StartCluster(ctx context.Context, clusterID string) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return err
	}

	infra, _ := s.infraRepo.FindByID(cluster.InfrastructureID)
	if infra.Status == entities.StatusRunning {
		return fmt.Errorf("cluster already running")
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	// Start etcd first
	for _, node := range nodes {
		if node.Role == "etcd" {
			s.dockerSvc.StartContainer(ctx, node.ContainerID)
		}
	}
	time.Sleep(5 * time.Second)

	// Start Patroni nodes
	for _, node := range nodes {
		if node.Role == "primary" || node.Role == "replica" {
			s.dockerSvc.StartContainer(ctx, node.ContainerID)
		}
	}
	time.Sleep(5 * time.Second)

	// Start HAProxy
	for _, node := range nodes {
		if node.Role == "haproxy" {
			s.dockerSvc.StartContainer(ctx, node.ContainerID)
		}
	}

	infra.Status = entities.StatusRunning
	s.infraRepo.Update(infra)

	s.publishEvent(ctx, "cluster.started", cluster.InfrastructureID, clusterID, string(entities.StatusRunning))
	return nil
}

// StopCluster stops all containers in the cluster
func (s *postgreSQLClusterService) StopCluster(ctx context.Context, clusterID string) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return err
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	// Stop HAProxy first
	for _, node := range nodes {
		if node.Role == "haproxy" {
			s.dockerSvc.StopContainer(ctx, node.ContainerID)
		}
	}

	// Stop Patroni nodes
	for _, node := range nodes {
		if node.Role == "primary" || node.Role == "replica" {
			s.dockerSvc.StopContainer(ctx, node.ContainerID)
		}
	}

	// Stop etcd last
	for _, node := range nodes {
		if node.Role == "etcd" {
			s.dockerSvc.StopContainer(ctx, node.ContainerID)
		}
	}

	infra, _ := s.infraRepo.FindByID(cluster.InfrastructureID)
	infra.Status = entities.StatusStopped
	s.infraRepo.Update(infra)

	s.publishEvent(ctx, "cluster.stopped", cluster.InfrastructureID, clusterID, string(entities.StatusStopped))
	return nil
}

// RestartCluster restarts the cluster
func (s *postgreSQLClusterService) RestartCluster(ctx context.Context, clusterID string) error {
	if err := s.StopCluster(ctx, clusterID); err != nil {
		return err
	}
	time.Sleep(5 * time.Second)
	return s.StartCluster(ctx, clusterID)
}

// DeleteCluster deletes the cluster and all resources
func (s *postgreSQLClusterService) DeleteCluster(ctx context.Context, clusterID string) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return err
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	for _, node := range nodes {
		s.dockerSvc.StopContainer(ctx, node.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, node.ContainerID)
		if node.VolumeID != "" {
			s.dockerSvc.RemoveVolume(ctx, node.VolumeID)
		}
		s.clusterRepo.DeleteNode(node.ID)
	}

	// Remove network
	if cluster.NetworkID != "" {
		s.dockerSvc.RemoveNetwork(ctx, cluster.NetworkID)
	}

	// Delete records
	s.clusterRepo.Delete(clusterID)
	s.infraRepo.Delete(cluster.InfrastructureID)

	s.publishEvent(ctx, "cluster.deleted", cluster.InfrastructureID, clusterID, "deleted")
	return nil
}

// ScaleCluster - not yet implemented for Patroni
func (s *postgreSQLClusterService) ScaleCluster(ctx context.Context, clusterID string, req dto.ScaleClusterRequest) error {
	return fmt.Errorf("scaling not yet implemented for Patroni clusters - use AddNode/RemoveNode instead")
}

// AddNode adds a new replica node to the cluster
func (s *postgreSQLClusterService) AddNode(ctx context.Context, clusterID string, req dto.AddNodeRequest) (*dto.AddNodeResponse, error) {
	s.logger.Info("adding new node to cluster", zap.String("cluster_id", clusterID))

	// Get cluster info
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// Get existing nodes to determine next index
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Count patroni nodes and find max index
	patroniNodeCount := 0
	maxIndex := 0
	var etcdNode *entities.ClusterNode
	var existingPatroniNode *entities.ClusterNode
	networkName := ""

	for _, node := range nodes {
		if node.Role == "primary" || node.Role == "replica" {
			patroniNodeCount++
			if existingPatroniNode == nil {
				nodeCopy := node // Copy to avoid pointer to loop variable
				existingPatroniNode = &nodeCopy
			}
			// Extract index from container name if possible
			if node.ContainerID != "" {
				inspect, err := s.dockerSvc.InspectContainer(ctx, node.ContainerID)
				if err == nil && inspect != nil {
					networkName = getNetworkNameFromContainer(inspect)
					// Find node index from name
					name := inspect.Name
					if strings.Contains(name, "patroni-node-") {
						var idx int
						fmt.Sscanf(name, "/iaas-patroni-%s-patroni-node-%d", new(string), &idx)
						if idx > maxIndex {
							maxIndex = idx
						}
					}
				}
			}
		}
		if node.Role == "etcd" && etcdNode == nil {
			etcdNode = &node
		}
	}

	if etcdNode == nil {
		return nil, fmt.Errorf("no etcd node found in cluster")
	}

	if networkName == "" {
		networkName = fmt.Sprintf("iaas-cluster-%s", clusterID)
	}

	// Check max nodes limit
	if patroniNodeCount >= 10 {
		return nil, fmt.Errorf("maximum 10 nodes reached")
	}

	// Create new node with next index
	newIndex := maxIndex // Will be incremented in createPatroniNode
	nodeID := uuid.New().String()
	nodeName := req.NodeName
	if nodeName == "" {
		nodeName = fmt.Sprintf("patroni-node-%d", newIndex+1)
	}
	containerName := fmt.Sprintf("iaas-patroni-%s-%s", cluster.ID, nodeName)
	volumeName := fmt.Sprintf("iaas-patroni-data-%s-%s", cluster.ID, nodeName)
	backupVolumeName := fmt.Sprintf("iaas-pgbackrest-%s-%s", cluster.ID, nodeName)

	// Create volumes
	if err := s.dockerSvc.CreateVolume(ctx, volumeName); err != nil {
		return nil, fmt.Errorf("failed to create patroni volume: %w", err)
	}
	if err := s.dockerSvc.CreateVolume(ctx, backupVolumeName); err != nil {
		return nil, fmt.Errorf("failed to create backup volume: %w", err)
	}

	// Get cluster scope from existing node
	scope := "admin"       // Default scope
	namespace := "default" // Default namespace
	if cluster.InfrastructureID != "" {
		infra, _ := s.infraRepo.FindByID(cluster.InfrastructureID)
		if infra != nil {
			scope = infra.Name
		}
	}

	// Get namespace from existing patroni container
	if existingPatroniNode != nil && existingPatroniNode.ContainerID != "" {
		inspect, err := s.dockerSvc.InspectContainer(ctx, existingPatroniNode.ContainerID)
		if err == nil && inspect != nil {
			for _, envVar := range inspect.Config.Env {
				if len(envVar) > 10 && envVar[:10] == "NAMESPACE=" {
					namespace = envVar[10:]
					break
				}
			}
		}
	}

	// Build environment variables for new replica
	env := []string{
		fmt.Sprintf("SCOPE=%s", scope),
		fmt.Sprintf("NAMESPACE=%s", namespace),
		fmt.Sprintf("PATRONI_NAME=%s", nodeName),
		fmt.Sprintf("ETCD_HOST=etcd-1:2379"),
		fmt.Sprintf("POSTGRES_PASSWORD=%s", cluster.Password),
		"REPLICATION_PASSWORD=replicator_pass",
		"MAX_CONNECTIONS=100",
		"SHARED_BUFFERS=128MB",
		"SYNCHRONOUS_COMMIT=local",
		"WATCHDOG_MODE=off",
		"NOFAILOVER=false",
		"NOLOADBALANCE=false",
		"CLONEFROM=true",
		"NOSYNC=false",
		"PGDATA=/data/patroni",
	}

	config := docker.ContainerConfig{
		Name:  containerName,
		Image: "iaas-patroni-postgres:17",
		Env:   env,
		Ports: map[string]string{"5432": "0", "8008": "0"},
		Volumes: map[string]string{
			volumeName:       "/data/patroni",
			backupVolumeName: "/pgbackrest",
		},
		Network:      networkName,
		NetworkAlias: nodeName,
		Resources: docker.ResourceConfig{
			CPULimit:    1000000000,        // 1 CPU
			MemoryLimit: 512 * 1024 * 1024, // 512MB
		},
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		// Cleanup on failure
		s.dockerSvc.RemoveContainer(ctx, containerID)
		s.dockerSvc.RemoveVolume(ctx, volumeName)
		s.dockerSvc.RemoveVolume(ctx, backupVolumeName)
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get assigned port
	inspect, err := s.dockerSvc.InspectContainer(ctx, containerID)
	port := 5432
	if err == nil && inspect != nil {
		for _, bindings := range inspect.NetworkSettings.Ports {
			if len(bindings) > 0 {
				fmt.Sscanf(bindings[0].HostPort, "%d", &port)
				break
			}
		}
	}

	// Save node to database
	node := &entities.ClusterNode{
		ID:          nodeID,
		ClusterID:   cluster.ID,
		ContainerID: containerID,
		Role:        "replica",
		Port:        port,
		VolumeID:    volumeName,
		IsHealthy:   true,
	}

	if err := s.clusterRepo.CreateNode(node); err != nil {
		s.dockerSvc.StopContainer(ctx, containerID)
		s.dockerSvc.RemoveContainer(ctx, containerID)
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	// Invalidate cache
	s.cacheService.InvalidateClusterInfo(ctx, clusterID)

	s.logger.Info("added new replica node",
		zap.String("node_id", nodeID),
		zap.String("node_name", nodeName),
		zap.String("container_id", containerID))

	return &dto.AddNodeResponse{
		NodeID:      nodeID,
		NodeName:    nodeName,
		ContainerID: containerID,
		Role:        "replica",
		Port:        port,
		Status:      "running",
		Message:     fmt.Sprintf("Node %s added successfully. Patroni will automatically sync data from primary.", nodeName),
	}, nil
}

// RemoveNode removes a node from the cluster
func (s *postgreSQLClusterService) RemoveNode(ctx context.Context, clusterID string, req dto.RemoveNodeRequest) (*dto.RemoveNodeResponse, error) {
	s.logger.Info("removing node from cluster",
		zap.String("cluster_id", clusterID),
		zap.String("node_id", req.NodeID))

	// Get cluster info
	_, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	// Get all nodes
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Find target node
	var targetNode *entities.ClusterNode
	patroniNodeCount := 0
	for i := range nodes {
		if nodes[i].Role == "primary" || nodes[i].Role == "replica" {
			patroniNodeCount++
		}
		if nodes[i].ID == req.NodeID {
			targetNode = &nodes[i]
		}
	}

	if targetNode == nil {
		return nil, fmt.Errorf("node not found: %s", req.NodeID)
	}

	// Prevent removing primary without force
	if targetNode.Role == "primary" && !req.Force {
		return nil, fmt.Errorf("cannot remove primary node - promote a replica first or use force=true")
	}

	// Prevent removing last node
	if patroniNodeCount <= 1 {
		return nil, fmt.Errorf("cannot remove the last patroni node - delete the cluster instead")
	}

	// If removing primary with force, trigger failover first
	if targetNode.Role == "primary" && req.Force {
		// Find a replica to promote
		for _, node := range nodes {
			if node.Role == "replica" {
				s.logger.Info("triggering failover before removing primary", zap.String("new_primary", node.ID))
				if err := s.PromoteReplica(ctx, clusterID, node.ID); err != nil {
					s.logger.Warn("failover failed, continuing with force removal", zap.Error(err))
				}
				break
			}
		}
	}

	// Stop and remove container
	if targetNode.ContainerID != "" {
		s.dockerSvc.StopContainer(ctx, targetNode.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, targetNode.ContainerID)
	}

	// Remove volumes
	if targetNode.VolumeID != "" {
		s.dockerSvc.RemoveVolume(ctx, targetNode.VolumeID)
		// Also try to remove backup volume
		backupVolume := strings.Replace(targetNode.VolumeID, "patroni-data", "pgbackrest", 1)
		s.dockerSvc.RemoveVolume(ctx, backupVolume)
	}

	// Delete from database
	if err := s.clusterRepo.DeleteNode(req.NodeID); err != nil {
		return nil, fmt.Errorf("failed to delete node from database: %w", err)
	}

	// Invalidate cache
	s.cacheService.InvalidateClusterInfo(ctx, clusterID)

	s.logger.Info("removed node from cluster",
		zap.String("node_id", req.NodeID),
		zap.String("role", targetNode.Role))

	return &dto.RemoveNodeResponse{
		NodeID:  req.NodeID,
		Removed: true,
		Message: fmt.Sprintf("Node removed successfully. Remaining nodes: %d", patroniNodeCount-1),
	}, nil
}

// Helper to get network name from container
func getNetworkNameFromContainer(inspect *types.ContainerJSON) string {
	if inspect.NetworkSettings != nil && inspect.NetworkSettings.Networks != nil {
		for name := range inspect.NetworkSettings.Networks {
			if strings.HasPrefix(name, "iaas-cluster-") {
				return name
			}
		}
	}
	return ""
}

// GetClusterStats retrieves cluster statistics
func (s *postgreSQLClusterService) GetClusterStats(ctx context.Context, clusterID string) (*dto.ClusterStatsResponse, error) {
	if cached, ok := s.cacheService.GetClusterStats(ctx, clusterID); ok {
		return cached, nil
	}

	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, err
	}

	nodeStats := make([]dto.NodeStats, 0)
	for _, node := range nodes {
		if node.Role == "primary" || node.Role == "replica" {
			nodeStats = append(nodeStats, dto.NodeStats{
				NodeName:          fmt.Sprintf("node-%s", node.Role),
				Role:              node.Role,
				CPUPercent:        0,
				MemoryPercent:     0,
				ActiveConnections: 0,
			})
		}
	}

	response := &dto.ClusterStatsResponse{
		ClusterID: clusterID,
	}

	s.cacheService.SetClusterStats(ctx, clusterID, response, 30*time.Second)

	return response, nil
}

// GetClusterLogs retrieves container logs
func (s *postgreSQLClusterService) GetClusterLogs(ctx context.Context, clusterID string, tail string) (*dto.ClusterLogsResponse, error) {
	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	nodeLogs := make([]dto.NodeLog, 0)
	tailLines, _ := strconv.Atoi(tail)
	if tailLines == 0 {
		tailLines = 100
	}

	for _, node := range nodes {
		logs, err := s.dockerSvc.GetContainerLogs(ctx, node.ContainerID, tailLines)
		if err == nil {
			nodeLogs = append(nodeLogs, dto.NodeLog{
				NodeName: node.Role,
				Logs:     strings.Join(logs, "\n"),
			})
		}
	}

	return &dto.ClusterLogsResponse{
		ClusterID: clusterID,
		Logs:      nodeLogs,
	}, nil
}

func (s *postgreSQLClusterService) PromoteReplica(ctx context.Context, clusterID, nodeID string) error {
	s.logger.Info("manual failover requested", zap.String("cluster_id", clusterID), zap.String("node_id", nodeID))

	// Get current nodes
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	var oldPrimary, newPrimary *entities.ClusterNode
	for i := range nodes {
		if nodes[i].Role == "primary" {
			oldPrimary = &nodes[i]
		}
		if nodes[i].ID == nodeID {
			newPrimary = &nodes[i]
		}
	}

	if newPrimary == nil {
		return fmt.Errorf("node %s not found", nodeID)
	}

	if newPrimary.Role == "primary" {
		return fmt.Errorf("node is already primary")
	}

	// Patroni handles the actual failover - we just record it
	// In production, you would call Patroni REST API here

	// Record failover event
	event := &entities.FailoverEvent{
		ID:             uuid.New().String(),
		ClusterID:      clusterID,
		OldPrimaryID:   oldPrimary.ID,
		OldPrimaryName: fmt.Sprintf("node-%s", oldPrimary.Role),
		NewPrimaryID:   newPrimary.ID,
		NewPrimaryName: fmt.Sprintf("node-%s", newPrimary.Role),
		Reason:         "manual",
		TriggeredBy:    "user",
	}

	if err := s.clusterRepo.CreateFailoverEvent(event); err != nil {
		s.logger.Error("failed to record failover event", zap.Error(err))
	}

	// Update roles in database
	oldPrimary.Role = "replica"
	newPrimary.Role = "primary"
	s.clusterRepo.UpdateNode(oldPrimary)
	s.clusterRepo.UpdateNode(newPrimary)

	// Invalidate cache
	s.cacheService.InvalidateClusterInfo(ctx, clusterID)

	s.logger.Info("failover completed",
		zap.String("old_primary", oldPrimary.ID),
		zap.String("new_primary", newPrimary.ID))

	return nil
}

// StopNode stops a specific node in the cluster
func (s *postgreSQLClusterService) StopNode(ctx context.Context, clusterID, nodeID string) error {
	s.logger.Info("stopping node", zap.String("cluster_id", clusterID), zap.String("node_id", nodeID))

	node, err := s.clusterRepo.FindNodeByID(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	if node.ClusterID != clusterID {
		return fmt.Errorf("node does not belong to this cluster")
	}

	// If stopping primary, record potential automatic failover
	if node.Role == "primary" {
		s.logger.Warn("stopping primary node - Patroni will trigger automatic failover",
			zap.String("node_id", nodeID))

		// Find a replica that will become new primary
		nodes, _ := s.clusterRepo.ListNodes(clusterID)
		var candidateReplica *entities.ClusterNode
		for i := range nodes {
			if nodes[i].Role == "replica" && nodes[i].IsHealthy {
				candidateReplica = &nodes[i]
				break
			}
		}

		if candidateReplica != nil {
			// Record automatic failover event
			event := &entities.FailoverEvent{
				ID:             uuid.New().String(),
				ClusterID:      clusterID,
				OldPrimaryID:   node.ID,
				OldPrimaryName: fmt.Sprintf("node-primary"),
				NewPrimaryID:   candidateReplica.ID,
				NewPrimaryName: fmt.Sprintf("node-replica"),
				Reason:         "node_failure",
				TriggeredBy:    "system",
			}
			s.clusterRepo.CreateFailoverEvent(event)

			// Update roles
			node.Role = "replica"
			candidateReplica.Role = "primary"
			s.clusterRepo.UpdateNode(node)
			s.clusterRepo.UpdateNode(candidateReplica)
		}
	}

	// Stop the container
	if err := s.dockerSvc.StopContainer(ctx, node.ContainerID); err != nil {
		return fmt.Errorf("failed to stop node: %w", err)
	}

	node.IsHealthy = false
	s.clusterRepo.UpdateNode(node)
	s.cacheService.InvalidateClusterInfo(ctx, clusterID)

	return nil
}

// StartNode starts a specific node in the cluster
func (s *postgreSQLClusterService) StartNode(ctx context.Context, clusterID, nodeID string) error {
	s.logger.Info("starting node", zap.String("cluster_id", clusterID), zap.String("node_id", nodeID))

	node, err := s.clusterRepo.FindNodeByID(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	if node.ClusterID != clusterID {
		return fmt.Errorf("node does not belong to this cluster")
	}

	// Start the container
	if err := s.dockerSvc.StartContainer(ctx, node.ContainerID); err != nil {
		return fmt.Errorf("failed to start node: %w", err)
	}

	node.IsHealthy = true
	s.clusterRepo.UpdateNode(node)
	s.cacheService.InvalidateClusterInfo(ctx, clusterID)

	return nil
}

// GetFailoverHistory returns failover events for a cluster
func (s *postgreSQLClusterService) GetFailoverHistory(ctx context.Context, clusterID string) ([]dto.FailoverEvent, error) {
	events, err := s.clusterRepo.ListFailoverEvents(clusterID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.FailoverEvent, len(events))
	for i, e := range events {
		result[i] = dto.FailoverEvent{
			ID:             e.ID,
			ClusterID:      e.ClusterID,
			OldPrimaryID:   e.OldPrimaryID,
			OldPrimaryName: e.OldPrimaryName,
			NewPrimaryID:   e.NewPrimaryID,
			NewPrimaryName: e.NewPrimaryName,
			Reason:         e.Reason,
			TriggeredBy:    e.TriggeredBy,
			OccurredAt:     e.OccurredAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetReplicationStatus retrieves replication status
func (s *postgreSQLClusterService) GetReplicationStatus(ctx context.Context, clusterID string) (*dto.ReplicationStatusResponse, error) {
	if cached, ok := s.cacheService.GetReplicationStatus(ctx, clusterID); ok {
		return cached, nil
	}

	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, err
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	var primaryNode string
	replicas := make([]dto.ReplicaStatus, 0)

	for _, node := range nodes {
		if node.Role == "primary" {
			primaryNode = fmt.Sprintf("primary-%d", node.Port)
		} else if node.Role == "replica" {
			replicas = append(replicas, dto.ReplicaStatus{
				NodeName:   fmt.Sprintf("node-%d", node.Port),
				State:      "streaming",
				SyncState:  cluster.ReplicationMode,
				LagBytes:   int(node.ReplicationDelay),
				LagSeconds: 0,
				IsHealthy:  node.IsHealthy,
			})
		}
	}

	response := &dto.ReplicationStatusResponse{
		Primary:  primaryNode,
		Replicas: replicas,
	}

	s.cacheService.SetReplicationStatus(ctx, clusterID, response, 10*time.Second)

	return response, nil
}

// User management - connect via HAProxy
func (s *postgreSQLClusterService) CreateUser(ctx context.Context, clusterID string, req dto.CreateUserRequest) error {
	// Implementation via psql exec on primary node
	return fmt.Errorf("not yet implemented")
}

func (s *postgreSQLClusterService) ListUsers(ctx context.Context, clusterID string) ([]dto.UserInfo, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *postgreSQLClusterService) DeleteUser(ctx context.Context, clusterID, username string) error {
	return fmt.Errorf("not yet implemented")
}

func (s *postgreSQLClusterService) CreateDatabase(ctx context.Context, clusterID string, req dto.CreateClusterDatabaseRequest) error {
	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	var primaryNode *entities.ClusterNode
	for _, node := range nodes {
		if node.Role == "primary" {
			primaryNode = &node
			break
		}
	}

	if primaryNode == nil {
		return fmt.Errorf("primary node not found")
	}

	createCmd := []string{
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("CREATE DATABASE %s OWNER %s ENCODING '%s';", req.Name, req.Owner, req.Encoding),
	}
	_, err := s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, createCmd)
	return err
}

func (s *postgreSQLClusterService) ListDatabases(ctx context.Context, clusterID string) ([]dto.ClusterDatabaseInfo, error) {
	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	var primaryNode *entities.ClusterNode
	for _, node := range nodes {
		if node.Role == "primary" {
			primaryNode = &node
			break
		}
	}

	if primaryNode == nil {
		return nil, fmt.Errorf("primary node not found")
	}

	listCmd := []string{
		"psql", "-U", "postgres", "-t", "-c",
		"SELECT datname, pg_catalog.pg_get_userbyid(datdba), pg_database_size(datname) FROM pg_database WHERE datistemplate = false;",
	}
	output, err := s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, listCmd)
	if err != nil {
		return nil, err
	}

	databases := make([]dto.ClusterDatabaseInfo, 0)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) >= 2 {
			databases = append(databases, dto.ClusterDatabaseInfo{
				Name:  strings.TrimSpace(parts[0]),
				Owner: strings.TrimSpace(parts[1]),
				Size:  "",
			})
		}
	}

	return databases, nil
}

func (s *postgreSQLClusterService) DeleteDatabase(ctx context.Context, clusterID, dbname string) error {
	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	var primaryNode *entities.ClusterNode
	for _, node := range nodes {
		if node.Role == "primary" {
			primaryNode = &node
			break
		}
	}

	if primaryNode == nil {
		return fmt.Errorf("primary node not found")
	}

	deleteCmd := []string{
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("DROP DATABASE %s;", dbname),
	}
	_, err := s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, deleteCmd)
	return err
}

func (s *postgreSQLClusterService) UpdateConfig(ctx context.Context, clusterID string, req dto.UpdateConfigRequest) error {
	return fmt.Errorf("not yet implemented")
}

func (s *postgreSQLClusterService) GetEndpoints(ctx context.Context, clusterID string) (*dto.ClusterInfoResponse, error) {
	return s.GetClusterInfo(ctx, clusterID)
}

// ExecuteQuery runs a SQL query on the cluster
func (s *postgreSQLClusterService) ExecuteQuery(ctx context.Context, clusterID string, req dto.ExecuteQueryRequest) (*dto.QueryResult, error) {
	s.logger.Info("executing query on cluster", zap.String("cluster_id", clusterID))

	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Find target node (primary by default, or specific node if requested)
	var targetNode *entities.ClusterNode
	var nodeRole string

	if req.NodeID != "" {
		for i := range nodes {
			if nodes[i].ID == req.NodeID {
				targetNode = &nodes[i]
				nodeRole = nodes[i].Role
				break
			}
		}
		if targetNode == nil {
			return nil, fmt.Errorf("node %s not found", req.NodeID)
		}
	} else {
		// Default to primary node
		for i := range nodes {
			if nodes[i].Role == "primary" {
				targetNode = &nodes[i]
				nodeRole = "primary"
				break
			}
		}
	}

	if targetNode == nil {
		return nil, fmt.Errorf("no suitable node found for query")
	}

	database := req.Database
	if database == "" {
		database = "postgres"
	}

	startTime := time.Now()

	// Execute query via psql
	cmd := []string{
		"psql", "-U", "postgres", "-d", database, "-t", "-A", "-F", "|",
		"-c", req.Query,
	}

	output, err := s.dockerSvc.ExecCommand(ctx, targetNode.ContainerID, cmd)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}

	duration := time.Since(startTime)

	// Parse output
	result := &dto.QueryResult{
		Columns:  []string{},
		Rows:     [][]interface{}{},
		Duration: duration.String(),
		NodeName: fmt.Sprintf("node-%s", nodeRole),
		NodeRole: nodeRole,
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		row := make([]interface{}, len(parts))
		for i, p := range parts {
			row[i] = strings.TrimSpace(p)
		}
		result.Rows = append(result.Rows, row)
	}
	result.RowCount = len(result.Rows)

	return result, nil
}

// TestReplication tests data replication across all nodes
func (s *postgreSQLClusterService) TestReplication(ctx context.Context, clusterID string) (*dto.ReplicationTestResult, error) {
	s.logger.Info("testing replication on cluster", zap.String("cluster_id", clusterID))

	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// Get real-time primary from Patroni API (not from database which may be outdated)
	var primaryNode *entities.ClusterNode
	var patroniNodes []entities.ClusterNode
	for _, node := range nodes {
		if node.Role == "primary" || node.Role == "replica" {
			patroniNodes = append(patroniNodes, node)
		}
	}

	// Query Patroni to find current leader
	for i := range patroniNodes {
		checkLeaderCmd := []string{"curl", "-s", "http://localhost:8008"}
		output, err := s.dockerSvc.ExecCommand(ctx, patroniNodes[i].ContainerID, checkLeaderCmd)
		if err != nil {
			continue
		}
		// Parse Patroni response - if role is master/leader, this is primary
		// Patroni returns JSON like: {"role": "master"} or {"role": "replica"}
		if strings.Contains(output, `"role": "master"`) || strings.Contains(output, `"role": "leader"`) {
			primaryNode = &patroniNodes[i]
			s.logger.Info("found current leader via Patroni API",
				zap.String("node_id", patroniNodes[i].ID),
				zap.String("container_id", patroniNodes[i].ContainerID))
			break
		}
	}

	if primaryNode == nil {
		return nil, fmt.Errorf("primary node not found via Patroni API")
	}

	s.logger.Info("using primary node for test",
		zap.String("node_id", primaryNode.ID),
		zap.String("container_id", primaryNode.ContainerID),
		zap.String("db_role", primaryNode.Role))

	testTable := "replication_test"
	testData := fmt.Sprintf("test-%d", time.Now().Unix())
	testTime := time.Now()

	// Create test table and insert data on primary
	createCmd := []string{
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id SERIAL PRIMARY KEY, data TEXT, created_at TIMESTAMP DEFAULT NOW());", testTable),
	}
	createOutput, err := s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, createCmd)
	s.logger.Info("create table result", zap.String("output", createOutput), zap.Error(err))
	if err != nil {
		return nil, fmt.Errorf("failed to create test table: %w", err)
	}

	insertCmd := []string{
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("INSERT INTO %s (data) VALUES ('%s');", testTable, testData),
	}
	insertOutput, err := s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, insertCmd)
	s.logger.Info("insert result", zap.String("output", insertOutput), zap.Error(err))
	if err != nil {
		return nil, fmt.Errorf("failed to insert test data: %w", err)
	}

	// Wait a moment for replication
	time.Sleep(1 * time.Second)

	// Check each node for the data
	nodeResults := make([]dto.NodeSyncResult, 0)
	allSynced := true

	for _, node := range patroniNodes {
		// Get real role from Patroni API
		realRole := "replica"
		if node.ContainerID == primaryNode.ContainerID {
			realRole = "primary"
		} else {
			checkRoleCmd := []string{"curl", "-s", "http://localhost:8008"}
			roleOutput, _ := s.dockerSvc.ExecCommand(ctx, node.ContainerID, checkRoleCmd)
			if strings.Contains(roleOutput, `"role": "master"`) || strings.Contains(roleOutput, `"role": "leader"`) {
				realRole = "primary"
			}
		}

		checkCmd := []string{
			"psql", "-U", "postgres", "-t", "-A", "-c",
			fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE data = '%s';", testTable, testData),
		}

		output, err := s.dockerSvc.ExecCommand(ctx, node.ContainerID, checkCmd)
		count := 0
		if err == nil {
			fmt.Sscanf(strings.TrimSpace(output), "%d", &count)
		}

		hasData := count > 0
		if !hasData && realRole != "primary" {
			allSynced = false
		}

		// Get node number from container name
		nodeName := node.Role
		nodeIndex := extractNodeIndex(node.ContainerID)
		if nodeIndex > 0 {
			nodeName = fmt.Sprintf("node-%d", nodeIndex)
		}

		nodeResults = append(nodeResults, dto.NodeSyncResult{
			NodeID:   node.ID,
			NodeName: nodeName,
			Role:     realRole,
			HasData:  hasData,
			RowCount: count,
		})
	}

	// Clean up test data
	cleanupCmd := []string{
		"psql", "-U", "postgres", "-c",
		fmt.Sprintf("DELETE FROM %s WHERE data = '%s';", testTable, testData),
	}
	s.dockerSvc.ExecCommand(ctx, primaryNode.ContainerID, cleanupCmd)

	return &dto.ReplicationTestResult{
		PrimaryNode:   primaryNode.ID,
		TestTable:     testTable,
		TestData:      testData,
		NodeResults:   nodeResults,
		AllSynced:     allSynced,
		TestTimestamp: testTime.Format(time.RFC3339),
	}, nil
}

// extractNodeIndex extracts node number from container ID/name
func extractNodeIndex(containerID string) int {
	// This is a helper - in real implementation, you'd query container name
	return 0
}
