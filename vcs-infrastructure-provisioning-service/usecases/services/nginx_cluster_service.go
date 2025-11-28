package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"net"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/kafka"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
)


// INginxClusterService interface for Nginx cluster operations
type INginxClusterService interface {
	CreateCluster(ctx context.Context, userID string, req dto.CreateNginxClusterRequest) (*dto.NginxClusterInfoResponse, error)
	GetClusterInfo(ctx context.Context, clusterID string) (*dto.NginxClusterInfoResponse, error)
	DeleteCluster(ctx context.Context, clusterID string) error
	StartCluster(ctx context.Context, clusterID string) error
	StopCluster(ctx context.Context, clusterID string) error
	RestartCluster(ctx context.Context, clusterID string) error

	// Node operations
	AddNode(ctx context.Context, clusterID string, req dto.AddNginxNodeRequest) (*dto.NginxNodeInfo, error)
	RemoveNode(ctx context.Context, clusterID, nodeID string) error
	GetNodeInfo(ctx context.Context, nodeID string) (*dto.NginxNodeInfo, error)

	// Configuration
	UpdateClusterConfig(ctx context.Context, clusterID string, req dto.UpdateNginxClusterConfigRequest) error
	SyncConfig(ctx context.Context, clusterID string) error

	// Upstreams
	AddUpstream(ctx context.Context, clusterID string, req dto.AddNginxUpstreamRequest) error
	UpdateUpstream(ctx context.Context, clusterID, upstreamID string, req dto.UpdateNginxUpstreamRequest) error
	DeleteUpstream(ctx context.Context, clusterID, upstreamID string) error
	ListUpstreams(ctx context.Context, clusterID string) ([]dto.UpstreamInfo, error)

	// Server blocks
	AddServerBlock(ctx context.Context, clusterID string, req dto.AddNginxServerBlockRequest) error
	DeleteServerBlock(ctx context.Context, clusterID, blockID string) error
	ListServerBlocks(ctx context.Context, clusterID string) ([]dto.ServerBlockInfo, error)

	// Health & Monitoring
	GetClusterHealth(ctx context.Context, clusterID string) (*dto.NginxClusterHealthResponse, error)
	GetClusterMetrics(ctx context.Context, clusterID string) (*dto.NginxClusterMetricsResponse, error)
	TestConnection(ctx context.Context, clusterID string) (*dto.TestNginxConnectionResponse, error)
	GetConnectionInfo(ctx context.Context, clusterID string) (*dto.NginxConnectionInfoResponse, error)

	// Failover
	TriggerFailover(ctx context.Context, clusterID string, req dto.TriggerNginxFailoverRequest) (*dto.NginxFailoverResponse, error)
	GetFailoverHistory(ctx context.Context, clusterID string) (*dto.NginxFailoverHistoryResponse, error)
}

type nginxClusterService struct {
	infraRepo     repositories.IInfrastructureRepository
	clusterRepo   repositories.INginxClusterRepository
	dockerSvc     docker.IDockerService
	kafkaProducer kafka.IKafkaProducer
	logger        logger.ILogger
}

// NewNginxClusterService creates a new Nginx cluster service
func NewNginxClusterService(
	infraRepo repositories.IInfrastructureRepository,
	clusterRepo repositories.INginxClusterRepository,
	dockerSvc docker.IDockerService,
	kafkaProducer kafka.IKafkaProducer,
	logger logger.ILogger,
) INginxClusterService {
	return &nginxClusterService{
		infraRepo:     infraRepo,
		clusterRepo:   clusterRepo,
		dockerSvc:     dockerSvc,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

// CreateCluster creates a new Nginx HA cluster with Keepalived
func (s *nginxClusterService) CreateCluster(ctx context.Context, userID string, req dto.CreateNginxClusterRequest) (*dto.NginxClusterInfoResponse, error) {
	s.logger.Info("creating Nginx HA cluster", zap.String("name", req.ClusterName), zap.Int("nodes", req.NodeCount))

	// Set defaults
	if req.LoadBalanceMode == "" {
		req.LoadBalanceMode = "round_robin"
	}
	if req.HTTPPort == 0 {
		req.HTTPPort = 80
	}
	if req.HTTPSPort == 0 && req.SSLEnabled {
		req.HTTPSPort = 443
	}
	if req.VRRPInterface == "" {
		req.VRRPInterface = "eth0"
	}
	if req.VRRPRouterID == 0 {
		req.VRRPRouterID = 51
	}
	if req.HealthCheckPath == "" {
		req.HealthCheckPath = "/health"
	}
	if req.HealthCheckInterval == 0 {
		req.HealthCheckInterval = 5
	}
	if req.WorkerConnections == 0 {
		req.WorkerConnections = 1024
	}
	if req.KeepaliveTimeout == 0 {
		req.KeepaliveTimeout = 65
	}
	if req.ClientMaxBodySize == "" {
		req.ClientMaxBodySize = "10m"
	}
	if req.ErrorLogLevel == "" {
		req.ErrorLogLevel = "warn"
	}
	if req.GzipLevel == 0 {
		req.GzipLevel = 6
	}
	if req.GzipMinLength == 0 {
		req.GzipMinLength = 1000
	}
	if req.GzipTypes == "" {
		req.GzipTypes = "text/plain text/css application/json application/javascript application/xml"
	}
	if req.SSLProtocols == "" {
		req.SSLProtocols = "TLSv1.2 TLSv1.3"
	}

	// Create infrastructure record
	infraID := uuid.New().String()
	infra := &entities.Infrastructure{
		ID:     infraID,
		Name:   req.ClusterName,
		Type:   entities.TypeNginx,
		Status: entities.StatusCreating,
		UserID: userID,
	}
	if err := s.infraRepo.Create(infra); err != nil {
		return nil, fmt.Errorf("failed to create infrastructure: %w", err)
	}

	// Create cluster record with all configuration
	clusterID := uuid.New().String()
	cluster := &entities.NginxCluster{
		ID:               clusterID,
		InfrastructureID: infraID,
		ClusterName:      req.ClusterName,
		NodeCount:        req.NodeCount,
		HTTPPort:         req.HTTPPort,
		HTTPSPort:        req.HTTPSPort,

		// HA Config
		VirtualIP:           req.VirtualIP,
		VRRPInterface:       req.VRRPInterface,
		VRRPRouterID:        req.VRRPRouterID,
		HealthCheckEnabled:  req.HealthCheckEnabled,
		HealthCheckPath:     req.HealthCheckPath,
		HealthCheckInterval: req.HealthCheckInterval,

		// Load Balancing
		LoadBalanceMode: req.LoadBalanceMode,

		// SSL
		SSLEnabled:        req.SSLEnabled,
		SSLCertificate:    req.SSLCertificate,
		SSLPrivateKey:     req.SSLPrivateKey,
		SSLProtocols:      req.SSLProtocols,
		SSLSessionTimeout: req.SSLSessionTimeout,

		// Performance
		WorkerProcesses:   req.WorkerProcesses,
		WorkerConnections: req.WorkerConnections,
		KeepaliveTimeout:  req.KeepaliveTimeout,
		ClientMaxBodySize: req.ClientMaxBodySize,

		// Logging
		AccessLogEnabled: req.AccessLogEnabled,
		ErrorLogLevel:    req.ErrorLogLevel,

		// Caching
		CacheEnabled: req.CacheEnabled,
		CachePath:    req.CachePath,
		CacheSize:    req.CacheSize,

		// Rate Limiting
		RateLimitEnabled:        req.RateLimitEnabled,
		RateLimitRequestsPerSec: req.RateLimitRequestsPerSec,
		RateLimitBurst:          req.RateLimitBurst,

		// Gzip
		GzipEnabled:   req.GzipEnabled,
		GzipLevel:     req.GzipLevel,
		GzipMinLength: req.GzipMinLength,
		GzipTypes:     req.GzipTypes,

		// Resources
		CPULimit:    req.CPUPerNode,
		MemoryLimit: req.MemoryPerNode,
	}
	if err := s.clusterRepo.Create(cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// Create dedicated network
	networkName := fmt.Sprintf("nginx-cluster-%s", clusterID[:8])
	networkID, err := s.dockerSvc.CreateNetwork(ctx, networkName)
	if err != nil {
		s.updateInfraStatus(infraID, entities.StatusFailed)
		return nil, fmt.Errorf("failed to create network: %w", err)
	}
	cluster.NetworkID = networkID
	s.clusterRepo.Update(cluster)

	// Create Nginx nodes
	s.logger.Info("creating Nginx nodes with Keepalived")
	for i := 0; i < req.NodeCount; i++ {
		priority := 100 - i // First node has highest priority (master)
		role := "backup"
		if i == 0 {
			role = "master"
		}

		node, err := s.createNginxNode(ctx, cluster, req, i, networkName, role, priority)
		if err != nil {
			s.logger.Error("failed to create nginx node", zap.Int("index", i), zap.Error(err))
			s.cleanup(ctx, cluster, networkID)
			return nil, fmt.Errorf("failed to create nginx node %d: %w", i, err)
		}

		if i == 0 {
			cluster.MasterNodeID = node.ID
			s.clusterRepo.Update(cluster)
		}

		// Wait between nodes
		if i < req.NodeCount-1 {
			time.Sleep(3 * time.Second)
		}
	}

	// Create upstreams if provided
	for _, upstream := range req.Upstreams {
		if err := s.createUpstream(ctx, clusterID, upstream); err != nil {
			s.logger.Warn("failed to create upstream", zap.String("name", upstream.Name), zap.Error(err))
		}
	}

	// Create server blocks if provided
	for _, block := range req.ServerBlocks {
		if err := s.createServerBlock(ctx, clusterID, block); err != nil {
			s.logger.Warn("failed to create server block", zap.String("name", block.ServerName), zap.Error(err))
		}
	}

	// Generate and apply nginx config
	if err := s.generateAndApplyConfig(ctx, cluster); err != nil {
		s.logger.Warn("failed to apply initial config", zap.Error(err))
	}

	// Update status
	infra.Status = entities.StatusRunning
	s.infraRepo.Update(infra)

	s.publishEvent(ctx, "nginx_cluster.created", infraID, clusterID, string(entities.StatusRunning))
	s.logger.Info("Nginx cluster created successfully", zap.String("cluster_id", clusterID))

	return s.GetClusterInfo(ctx, clusterID)
}

func (s *nginxClusterService) createNginxNode(ctx context.Context, cluster *entities.NginxCluster, req dto.CreateNginxClusterRequest, index int, networkName, role string, priority int) (*entities.NginxNode, error) {
	nodeID := uuid.New().String()
	nodeName := fmt.Sprintf("%s-nginx-%d", cluster.ClusterName, index+1)
	// tìm port phù hợp thay vì fix cứng
	httpPort, httpsPort := s.getNextAvailablePortForCluster(cluster)

	env := []string{
		fmt.Sprintf("NGINX_NODE_NAME=%s", nodeName),
		fmt.Sprintf("NGINX_NODE_ROLE=%s", role),
		fmt.Sprintf("KEEPALIVED_STATE=%s", strings.ToUpper(role)),
		fmt.Sprintf("KEEPALIVED_PRIORITY=%d", priority),
		"KEEPALIVED_INTERFACE=eth0",
		fmt.Sprintf("KEEPALIVED_ROUTER_ID=%d", cluster.VRRPRouterID),
		fmt.Sprintf("VIRTUAL_IP=%s", cluster.VirtualIP),
		fmt.Sprintf("HTTP_PORT=%d", cluster.HTTPPort),
	}

	ports := map[string]string{
		"80": fmt.Sprintf("%d", httpPort),
	}
	if httpsPort > 0 {
		ports["443"] = fmt.Sprintf("%d", httpsPort)
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, docker.ContainerConfig{
		Name:         nodeName,
		Image:        "nginx:alpine",
		Network:      networkName,
		NetworkAlias: nodeName,
		Env:          env,
		Ports:        ports,
		Resources: docker.ResourceConfig{
			CPULimit:    cluster.CPULimit,
			MemoryLimit: cluster.MemoryLimit,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get container IP
	containerInfo, err := s.dockerSvc.InspectContainer(ctx, containerID)
	ipAddress := ""
	if err == nil && containerInfo != nil && containerInfo.NetworkSettings != nil {
		ipAddress = containerInfo.NetworkSettings.IPAddress
	}

	// Create node record
	node := &entities.NginxNode{
		ID:          nodeID,
		ClusterID:   cluster.ID,
		Name:        nodeName,
		ContainerID: containerID,
		Role:        role,
		Priority:    priority,
		HTTPPort:    httpPort,
		HTTPSPort:   httpsPort,
		Status:      string(entities.StatusRunning),
		IPAddress:   ipAddress,
		IsHealthy:   true,
	}
	if err := s.clusterRepo.CreateNode(node); err != nil {
		return nil, fmt.Errorf("failed to save node: %w", err)
	}

	s.logger.Info("created nginx node", zap.String("node_id", nodeID), zap.String("role", role), zap.Int("priority", priority))
	return node, nil
}

// GetClusterInfo retrieves cluster information
func (s *nginxClusterService) GetClusterInfo(ctx context.Context, clusterID string) (*dto.NginxClusterInfoResponse, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	infra, err := s.infraRepo.FindByID(cluster.InfrastructureID)
	if err != nil {
		return nil, fmt.Errorf("infrastructure not found: %w", err)
	}

	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodeInfos := make([]dto.NginxNodeInfo, 0, len(nodes))
	var masterNode *dto.NginxNodeInfo
	runningCount := 0

	for _, node := range nodes {
		// Get real-time status from Docker
		nodeStatus := node.Status
		isHealthy := node.IsHealthy

		if node.ContainerID != "" {
			if containerInfo, err := s.dockerSvc.InspectContainer(ctx, node.ContainerID); err == nil {
				nodeStatus = containerInfo.State.Status
				isHealthy = (nodeStatus == "running")

				// Update node status in database if changed
				if nodeStatus != node.Status {
					node.Status = nodeStatus
					node.IsHealthy = isHealthy
					s.clusterRepo.UpdateNode(&node)
				}
			}
		}

		if nodeStatus == "running" {
			runningCount++
		}

		nodeInfo := dto.NginxNodeInfo{
			ID:          node.ID,
			Name:        node.Name,
			Role:        node.Role,
			Priority:    node.Priority,
			Status:      nodeStatus,
			ContainerID: node.ContainerID,
			IPAddress:   node.IPAddress,
			HTTPPort:    node.HTTPPort,
			HTTPSPort:   node.HTTPSPort,
			IsHealthy:   isHealthy,
			IsMaster:    node.Role == "master",
		}
		nodeInfos = append(nodeInfos, nodeInfo)
		if node.ID == cluster.MasterNodeID {
			masterNode = &nodeInfo
		}
	}

	// Update cluster status based on nodes
	clusterStatus := string(infra.Status)
	if runningCount == 0 {
		clusterStatus = "stopped"
	} else if runningCount < len(nodes) {
		clusterStatus = "degraded"
	} else {
		clusterStatus = "running"
	}

	// Update infrastructure status if changed
	if clusterStatus != string(infra.Status) {
		infra.Status = entities.InfrastructureStatus(clusterStatus)
		s.infraRepo.Update(infra)
	}

	// Get upstreams
	upstreams, _ := s.ListUpstreams(ctx, clusterID)

	// Get server blocks
	serverBlocks, _ := s.ListServerBlocks(ctx, clusterID)

	// Build endpoints
	endpoints := dto.NginxEndpoints{
		HTTPURL: fmt.Sprintf("http://localhost:%d", cluster.HTTPPort),
	}
	if cluster.HTTPSPort > 0 {
		endpoints.HTTPSURL = fmt.Sprintf("https://localhost:%d", cluster.HTTPSPort)
	}
	if cluster.VirtualIP != "" {
		endpoints.VirtualIP = cluster.VirtualIP
	}
	if masterNode != nil {
		endpoints.MasterNode = masterNode.Name
	}

	return &dto.NginxClusterInfoResponse{
		ID:               clusterID,
		InfrastructureID: cluster.InfrastructureID,
		ClusterName:      cluster.ClusterName,
		Status:           clusterStatus,
		NodeCount:        cluster.NodeCount,
		MasterNode:       masterNode,
		Nodes:            nodeInfos,
		VirtualIP:        cluster.VirtualIP,
		HTTPPort:         cluster.HTTPPort,
		HTTPSPort:        cluster.HTTPSPort,
		LoadBalanceMode:  cluster.LoadBalanceMode,
		SSLEnabled:       cluster.SSLEnabled,
		Upstreams:        upstreams,
		ServerBlocks:     serverBlocks,
		Endpoints:        endpoints,
		CreatedAt:        cluster.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        cluster.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// DeleteCluster removes the entire cluster
func (s *nginxClusterService) DeleteCluster(ctx context.Context, clusterID string) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	// Stop and remove all nodes
	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	for _, node := range nodes {
		if node.ContainerID != "" {
			s.dockerSvc.StopContainer(ctx, node.ContainerID)
			s.dockerSvc.RemoveContainer(ctx, node.ContainerID)
		}
		s.clusterRepo.DeleteNode(node.ID)
	}

	// Remove network
	if cluster.NetworkID != "" {
		s.dockerSvc.RemoveNetwork(ctx, cluster.NetworkID)
	}

	// Delete cluster and infrastructure
	s.clusterRepo.Delete(clusterID)
	s.infraRepo.Delete(cluster.InfrastructureID)

	s.publishEvent(ctx, "nginx_cluster.deleted", cluster.InfrastructureID, clusterID, "deleted")
	return nil
}

// StartCluster starts all nodes in the cluster
func (s *nginxClusterService) StartCluster(ctx context.Context, clusterID string) error {
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes {
		if err := s.dockerSvc.StartContainer(ctx, node.ContainerID); err != nil {
			s.logger.Error("failed to start node", zap.String("node_id", node.ID), zap.Error(err))
		} else {
			node.Status = string(entities.StatusRunning)
			s.clusterRepo.UpdateNode(&node)
		}
	}

	cluster, _ := s.clusterRepo.FindByID(clusterID)
	s.updateInfraStatus(cluster.InfrastructureID, entities.StatusRunning)
	return nil
}

// StopCluster stops all nodes in the cluster
func (s *nginxClusterService) StopCluster(ctx context.Context, clusterID string) error {
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	for _, node := range nodes {
		if err := s.dockerSvc.StopContainer(ctx, node.ContainerID); err != nil {
			s.logger.Error("failed to stop node", zap.String("node_id", node.ID), zap.Error(err))
		} else {
			node.Status = string(entities.StatusStopped)
			s.clusterRepo.UpdateNode(&node)
		}
	}

	cluster, _ := s.clusterRepo.FindByID(clusterID)
	s.updateInfraStatus(cluster.InfrastructureID, entities.StatusStopped)
	return nil
}

// RestartCluster restarts all nodes
func (s *nginxClusterService) RestartCluster(ctx context.Context, clusterID string) error {
	if err := s.StopCluster(ctx, clusterID); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return s.StartCluster(ctx, clusterID)
}

// AddNode adds a new node to the cluster
func (s *nginxClusterService) AddNode(ctx context.Context, clusterID string, req dto.AddNginxNodeRequest) (*dto.NginxNodeInfo, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	index := len(nodes)
	priority := req.Priority
	if priority == 0 {
		priority = 100 - index
	}

	node, err := s.createNginxNode(ctx, cluster, dto.CreateNginxClusterRequest{}, index, fmt.Sprintf("nginx-cluster-%s", clusterID[:8]), "backup", priority)
	if err != nil {
		return nil, err
	}

	cluster.NodeCount++
	s.clusterRepo.Update(cluster)

	return &dto.NginxNodeInfo{
		ID:          node.ID,
		Name:        node.Name,
		Role:        node.Role,
		Priority:    node.Priority,
		Status:      node.Status,
		ContainerID: node.ContainerID,
		IPAddress:   node.IPAddress,
		HTTPPort:    node.HTTPPort,
		HTTPSPort:   node.HTTPSPort,
		IsHealthy:   node.IsHealthy,
	}, nil
}

// RemoveNode removes a node from the cluster
func (s *nginxClusterService) RemoveNode(ctx context.Context, clusterID, nodeID string) error {
	node, err := s.clusterRepo.FindNodeByID(nodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	if node.Role == "master" {
		return fmt.Errorf("cannot remove master node, trigger failover first")
	}

	// Stop and remove container
	if node.ContainerID != "" {
		s.dockerSvc.StopContainer(ctx, node.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, node.ContainerID)
	}

	s.clusterRepo.DeleteNode(nodeID)

	// Update cluster node count
	cluster, _ := s.clusterRepo.FindByID(clusterID)
	cluster.NodeCount--
	s.clusterRepo.Update(cluster)

	return nil
}

// GetNodeInfo retrieves node information
func (s *nginxClusterService) GetNodeInfo(ctx context.Context, nodeID string) (*dto.NginxNodeInfo, error) {
	node, err := s.clusterRepo.FindNodeByID(nodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	return &dto.NginxNodeInfo{
		ID:          node.ID,
		Name:        node.Name,
		Role:        node.Role,
		Priority:    node.Priority,
		Status:      node.Status,
		ContainerID: node.ContainerID,
		IPAddress:   node.IPAddress,
		HTTPPort:    node.HTTPPort,
		HTTPSPort:   node.HTTPSPort,
		IsHealthy:   node.IsHealthy,
		IsMaster:    node.Role == "master",
	}, nil
}

// UpdateClusterConfig updates nginx configuration on all nodes
func (s *nginxClusterService) UpdateClusterConfig(ctx context.Context, clusterID string, req dto.UpdateNginxClusterConfigRequest) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	cluster.NginxConfig = req.NginxConfig
	s.clusterRepo.Update(cluster)

	if req.ReloadAll {
		return s.SyncConfig(ctx, clusterID)
	}
	return nil
}

// SyncConfig synchronizes configuration to all nodes following NGINX best practices
// Reference: https://docs.nginx.com/nginx/admin-guide/high-availability/configuration-sharing/
func (s *nginxClusterService) SyncConfig(ctx context.Context, clusterID string) error {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return fmt.Errorf("cluster not found: %w", err)
	}

	s.logger.Info("starting config synchronization", zap.String("cluster_id", clusterID))

	// Step 1: Validate config on primary (master) node first
	masterNode, err := s.getMasterNode(clusterID)
	if err != nil {
		return fmt.Errorf("failed to get master node: %w", err)
	}

	// Validate on master
	if err := s.validateNginxConfig(ctx, masterNode.ContainerID, cluster.NginxConfig); err != nil {
		s.logger.Error("config validation failed on master", zap.Error(err))
		return fmt.Errorf("config validation failed on master: %w", err)
	}

	s.logger.Info("config validated on master node", zap.String("master_node", masterNode.Name))

	// Step 2: Sync to all peer nodes
	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	successCount := 0
	failedNodes := []string{}

	for _, node := range nodes {
		if node.ID == masterNode.ID {
			// Master already validated, just apply
			if err := s.applyConfigToNode(ctx, node.ContainerID, cluster.NginxConfig); err != nil {
				s.logger.Error("failed to apply config to master", zap.String("node_id", node.ID), zap.Error(err))
				failedNodes = append(failedNodes, node.Name)
			} else {
				successCount++
			}
			continue
		}

		// For peer nodes: backup -> validate -> apply -> reload
		if err := s.syncConfigToNode(ctx, &node, cluster.NginxConfig); err != nil {
			s.logger.Error("failed to sync config to peer", zap.String("node_id", node.ID), zap.Error(err))
			failedNodes = append(failedNodes, node.Name)
		} else {
			successCount++
		}
	}

	s.logger.Info("config synchronization completed",
		zap.Int("success", successCount),
		zap.Int("failed", len(failedNodes)),
		zap.Strings("failed_nodes", failedNodes))

	if len(failedNodes) > 0 {
		return fmt.Errorf("config sync failed on %d nodes: %v", len(failedNodes), failedNodes)
	}

	return nil
}

// syncConfigToNode syncs config to a single peer node with backup and rollback
func (s *nginxClusterService) syncConfigToNode(ctx context.Context, node *entities.NginxNode, config string) error {
	containerID := node.ContainerID

	// Step 1: Backup current config
	backupPath := fmt.Sprintf("/etc/nginx/nginx.conf.backup.%d", time.Now().Unix())
	backupCmd := fmt.Sprintf("cp /etc/nginx/nginx.conf %s", backupPath)
	if _, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", backupCmd}); err != nil {
		return fmt.Errorf("failed to backup config: %w", err)
	}

	s.logger.Info("config backed up", zap.String("node", node.Name), zap.String("backup_path", backupPath))

	// Step 2: Write new config
	writeCmd := fmt.Sprintf("cat > /etc/nginx/nginx.conf << 'EOF'\n%s\nEOF", config)
	if _, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", writeCmd}); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Step 3: Validate new config
	if err := s.validateNginxConfig(ctx, containerID, config); err != nil {
		// Rollback on validation failure
		s.logger.Warn("validation failed, rolling back", zap.String("node", node.Name))
		rollbackCmd := fmt.Sprintf("cp %s /etc/nginx/nginx.conf", backupPath)
		s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", rollbackCmd})
		return fmt.Errorf("config validation failed, rolled back: %w", err)
	}

	// Step 4: Reload nginx
	if _, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"nginx", "-s", "reload"}); err != nil {
		// Rollback on reload failure
		s.logger.Warn("reload failed, rolling back", zap.String("node", node.Name))
		rollbackCmd := fmt.Sprintf("cp %s /etc/nginx/nginx.conf && nginx -s reload", backupPath)
		s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", rollbackCmd})
		return fmt.Errorf("nginx reload failed, rolled back: %w", err)
	}

	s.logger.Info("config synced successfully", zap.String("node", node.Name))
	return nil
}

// validateNginxConfig validates nginx configuration
func (s *nginxClusterService) validateNginxConfig(ctx context.Context, containerID string, config string) error {
	// Write config to temp file and test
	testCmd := fmt.Sprintf("cat > /tmp/nginx.conf.test << 'EOF'\n%s\nEOF && nginx -t -c /tmp/nginx.conf.test", config)
	output, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", testCmd})
	if err != nil {
		return fmt.Errorf("nginx -t failed: %w, output: %s", err, output)
	}
	return nil
}

// applyConfigToNode applies config without validation (already validated)
func (s *nginxClusterService) applyConfigToNode(ctx context.Context, containerID string, config string) error {
	writeCmd := fmt.Sprintf("cat > /etc/nginx/nginx.conf << 'EOF'\n%s\nEOF", config)
	if _, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"sh", "-c", writeCmd}); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if _, err := s.dockerSvc.ExecCommand(ctx, containerID, []string{"nginx", "-s", "reload"}); err != nil {
		return fmt.Errorf("nginx reload failed: %w", err)
	}

	return nil
}

// getMasterNode returns the master node of the cluster
func (s *nginxClusterService) getMasterNode(clusterID string) (*entities.NginxNode, error) {
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, err
	}

	for _, node := range nodes {
		if node.Role == "master" {
			return &node, nil
		}
	}

	return nil, fmt.Errorf("no master node found")
}

// generateAndApplyConfig generates default nginx config and applies to all nodes
func (s *nginxClusterService) generateAndApplyConfig(ctx context.Context, cluster *entities.NginxCluster) error {
	config := s.generateNginxConfig(cluster)
	cluster.NginxConfig = config
	s.clusterRepo.Update(cluster)

	return s.SyncConfig(ctx, cluster.ID)
}

// generateNginxConfig generates nginx configuration from cluster settings
func (s *nginxClusterService) generateNginxConfig(cluster *entities.NginxCluster) string {
	workerProcesses := "auto"
	if cluster.WorkerProcesses > 0 {
		workerProcesses = fmt.Sprintf("%d", cluster.WorkerProcesses)
	}

	// Build configuration
	config := fmt.Sprintf(`# Nginx Configuration - Generated by IaaS Platform
# Cluster: %s
# Generated at: %s

worker_processes %s;
error_log /var/log/nginx/error.log %s;
pid /var/run/nginx.pid;

events {
    worker_connections %d;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging Configuration
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for" '
                    'rt=$request_time uct="$upstream_connect_time" '
                    'uht="$upstream_header_time" urt="$upstream_response_time"';
`,
		cluster.ClusterName,
		time.Now().Format(time.RFC3339),
		workerProcesses,
		cluster.ErrorLogLevel,
		cluster.WorkerConnections,
	)

	// Access log
	if cluster.AccessLogEnabled {
		config += "    access_log /var/log/nginx/access.log main;\n"
	} else {
		config += "    access_log off;\n"
	}

	// Basic settings
	config += fmt.Sprintf(`
    # Performance Settings
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout %d;
    types_hash_max_size 2048;
    server_tokens off;
    client_max_body_size %s;

`, cluster.KeepaliveTimeout, cluster.ClientMaxBodySize)

	// Gzip compression
	if cluster.GzipEnabled {
		config += fmt.Sprintf(`    # Gzip Compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level %d;
    gzip_min_length %d;
    gzip_types %s;

`, cluster.GzipLevel, cluster.GzipMinLength, cluster.GzipTypes)
	}

	// Rate limiting
	if cluster.RateLimitEnabled {
		config += fmt.Sprintf(`    # Rate Limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=%dr/s;

`, cluster.RateLimitRequestsPerSec)
	}

	// Caching
	if cluster.CacheEnabled && cluster.CachePath != "" {
		config += fmt.Sprintf(`    # Proxy Cache
    proxy_cache_path %s levels=1:2 keys_zone=nginx_cache:10m max_size=%s inactive=60m use_temp_path=off;

`, cluster.CachePath, cluster.CacheSize)
	}

	// SSL settings if enabled
	if cluster.SSLEnabled {
		config += fmt.Sprintf(`    # SSL Settings
    ssl_protocols %s;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout %s;
    ssl_session_tickets off;

`, cluster.SSLProtocols, cluster.SSLSessionTimeout)
	}

	// Proxy settings
	config += `    # Proxy Settings
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Connection "";
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    proxy_buffering on;
    proxy_buffer_size 4k;
    proxy_buffers 8 4k;

`

	// Default server block
	config += `    # Default Server
    server {
        listen 80 default_server;
        server_name _;

        # Health check endpoint
        location /health {
            access_log off;
            return 200 "{\"status\":\"healthy\",\"cluster\":\"` + cluster.ClusterName + `\"}";
            add_header Content-Type application/json;
        }

        # Nginx status for monitoring
        location /nginx_status {
            stub_status on;
            access_log off;
            allow 127.0.0.1;
            allow 10.0.0.0/8;
            allow 172.16.0.0/12;
            allow 192.168.0.0/16;
            deny all;
        }

        # Root
        location / {
            root /usr/share/nginx/html;
            index index.html index.htm;
            try_files $uri $uri/ =404;
        }
`

	// Add rate limiting to default server if enabled
	if cluster.RateLimitEnabled {
		config += fmt.Sprintf(`
        # Apply rate limiting
        limit_req zone=api_limit burst=%d nodelay;
`, cluster.RateLimitBurst)
	}

	config += `    }
}
`
	return config
}

// createUpstream creates an upstream
func (s *nginxClusterService) createUpstream(ctx context.Context, clusterID string, req dto.CreateUpstreamRequest) error {
	upstreamID := uuid.New().String()
	upstream := &entities.NginxClusterUpstream{
		ID:          upstreamID,
		ClusterID:   clusterID,
		Name:        req.Name,
		Algorithm:   req.Algorithm,
		HealthCheck: req.HealthCheck,
		HealthPath:  req.HealthPath,
	}
	if err := s.clusterRepo.CreateUpstream(upstream); err != nil {
		return err
	}

	for _, srv := range req.Servers {
		server := &entities.NginxUpstreamServer{
			ID:          uuid.New().String(),
			UpstreamID:  upstreamID,
			Address:     srv.Address,
			Weight:      srv.Weight,
			MaxFails:    srv.MaxFails,
			FailTimeout: srv.FailTimeout,
			IsBackup:    srv.IsBackup,
		}
		s.clusterRepo.CreateUpstreamServer(server)
	}
	return nil
}

// createServerBlock creates a server block
func (s *nginxClusterService) createServerBlock(ctx context.Context, clusterID string, req dto.CreateServerBlockRequest) error {
	blockID := uuid.New().String()
	block := &entities.NginxServerBlock{
		ID:         blockID,
		ClusterID:  clusterID,
		ServerName: req.ServerName,
		ListenPort: req.ListenPort,
		SSLEnabled: req.SSLEnabled,
		RootPath:   req.RootPath,
	}
	if err := s.clusterRepo.CreateServerBlock(block); err != nil {
		return err
	}

	for _, loc := range req.Locations {
		headers, _ := json.Marshal(loc.ProxyHeaders)
		location := &entities.NginxLocation{
			ID:            uuid.New().String(),
			ServerBlockID: blockID,
			Path:          loc.Path,
			ProxyPass:     loc.ProxyPass,
			ProxyHeaders:  string(headers),
			CacheEnabled:  loc.CacheEnabled,
			RateLimit:     loc.RateLimit,
		}
		s.clusterRepo.CreateLocation(location)
	}
	return nil
}

// AddUpstream adds an upstream to the cluster
func (s *nginxClusterService) AddUpstream(ctx context.Context, clusterID string, req dto.AddNginxUpstreamRequest) error {
	return s.createUpstream(ctx, clusterID, dto.CreateUpstreamRequest{
		Name:        req.Name,
		Algorithm:   req.Algorithm,
		Servers:     req.Servers,
		HealthCheck: req.HealthCheck,
		HealthPath:  req.HealthPath,
	})
}

// UpdateUpstream updates an upstream
func (s *nginxClusterService) UpdateUpstream(ctx context.Context, clusterID, upstreamID string, req dto.UpdateNginxUpstreamRequest) error {
	upstream, err := s.clusterRepo.FindUpstreamByID(upstreamID)
	if err != nil {
		return fmt.Errorf("upstream not found: %w", err)
	}

	if req.Algorithm != "" {
		upstream.Algorithm = req.Algorithm
	}
	upstream.HealthCheck = req.HealthCheck
	if req.HealthPath != "" {
		upstream.HealthPath = req.HealthPath
	}
	s.clusterRepo.UpdateUpstream(upstream)

	// Update servers
	if len(req.Servers) > 0 {
		s.clusterRepo.DeleteUpstreamServersByUpstreamID(upstreamID)
		for _, srv := range req.Servers {
			server := &entities.NginxUpstreamServer{
				ID:          uuid.New().String(),
				UpstreamID:  upstreamID,
				Address:     srv.Address,
				Weight:      srv.Weight,
				MaxFails:    srv.MaxFails,
				FailTimeout: srv.FailTimeout,
				IsBackup:    srv.IsBackup,
			}
			s.clusterRepo.CreateUpstreamServer(server)
		}
	}
	return nil
}

// DeleteUpstream deletes an upstream
func (s *nginxClusterService) DeleteUpstream(ctx context.Context, clusterID, upstreamID string) error {
	s.clusterRepo.DeleteUpstreamServersByUpstreamID(upstreamID)
	return s.clusterRepo.DeleteUpstream(upstreamID)
}

// ListUpstreams lists all upstreams
func (s *nginxClusterService) ListUpstreams(ctx context.Context, clusterID string) ([]dto.UpstreamInfo, error) {
	upstreams, err := s.clusterRepo.ListUpstreams(clusterID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.UpstreamInfo, 0, len(upstreams))
	for _, u := range upstreams {
		servers, _ := s.clusterRepo.ListUpstreamServers(u.ID)
		backends := make([]dto.BackendServer, 0, len(servers))
		for _, srv := range servers {
			backends = append(backends, dto.BackendServer{
				Address: srv.Address,
				Weight:  srv.Weight,
			})
		}
		result = append(result, dto.UpstreamInfo{
			Name:     u.Name,
			Backends: backends,
			Policy:   u.Algorithm,
		})
	}
	return result, nil
}

// AddServerBlock adds a server block
func (s *nginxClusterService) AddServerBlock(ctx context.Context, clusterID string, req dto.AddNginxServerBlockRequest) error {
	return s.createServerBlock(ctx, clusterID, dto.CreateServerBlockRequest{
		ServerName: req.ServerName,
		ListenPort: req.ListenPort,
		SSLEnabled: req.SSLEnabled,
		RootPath:   req.RootPath,
		Locations:  req.Locations,
	})
}

// DeleteServerBlock deletes a server block
func (s *nginxClusterService) DeleteServerBlock(ctx context.Context, clusterID, blockID string) error {
	s.clusterRepo.DeleteLocationsByServerBlockID(blockID)
	return s.clusterRepo.DeleteServerBlock(blockID)
}

// ListServerBlocks lists all server blocks
func (s *nginxClusterService) ListServerBlocks(ctx context.Context, clusterID string) ([]dto.ServerBlockInfo, error) {
	blocks, err := s.clusterRepo.ListServerBlocks(clusterID)
	if err != nil {
		return nil, err
	}

	result := make([]dto.ServerBlockInfo, 0, len(blocks))
	for _, b := range blocks {
		locations, _ := s.clusterRepo.ListLocations(b.ID)
		locInfos := make([]dto.LocationInfo, 0, len(locations))
		for _, loc := range locations {
			var headers map[string]string
			json.Unmarshal([]byte(loc.ProxyHeaders), &headers)
			locInfos = append(locInfos, dto.LocationInfo{
				ID:           loc.ID,
				Path:         loc.Path,
				ProxyPass:    loc.ProxyPass,
				ProxyHeaders: headers,
				CacheEnabled: loc.CacheEnabled,
				RateLimit:    loc.RateLimit,
			})
		}
		result = append(result, dto.ServerBlockInfo{
			ID:         b.ID,
			ServerName: b.ServerName,
			ListenPort: b.ListenPort,
			SSLEnabled: b.SSLEnabled,
			Locations:  locInfos,
		})
	}
	return result, nil
}

// GetClusterHealth returns cluster health status
func (s *nginxClusterService) GetClusterHealth(ctx context.Context, clusterID string) (*dto.NginxClusterHealthResponse, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	nodes, _ := s.clusterRepo.ListNodes(clusterID)
	healthyCount := 0
	nodeHealth := make([]dto.NodeHealthInfo, 0, len(nodes))
	masterName := ""

	for _, node := range nodes {
		healthy := s.checkNodeHealth(ctx, &node)
		if healthy {
			healthyCount++
		}
		if node.Role == "master" {
			masterName = node.Name
		}

		nodeHealth = append(nodeHealth, dto.NodeHealthInfo{
			NodeID:           node.ID,
			NodeName:         node.Name,
			Role:             node.Role,
			IsHealthy:        healthy,
			NginxStatus:      node.Status,
			KeepalivedStatus: "running",
			LastCheck:        time.Now().Format(time.RFC3339),
		})
	}

	status := "healthy"
	if healthyCount == 0 {
		status = "unhealthy"
	} else if healthyCount < len(nodes) {
		status = "degraded"
	}

	return &dto.NginxClusterHealthResponse{
		ClusterID:    clusterID,
		ClusterName:  cluster.ClusterName,
		Status:       status,
		HealthyNodes: healthyCount,
		TotalNodes:   len(nodes),
		MasterNode:   masterName,
		VIPActive:    cluster.VirtualIP != "",
		NodeHealth:   nodeHealth,
	}, nil
}

// checkNodeHealth checks if a node is healthy
func (s *nginxClusterService) checkNodeHealth(ctx context.Context, node *entities.NginxNode) bool {
	if node.ContainerID == "" {
		return false
	}

	// Check if container is running
	info, err := s.dockerSvc.InspectContainer(ctx, node.ContainerID)
	if err != nil || info == nil || info.State == nil {
		return false
	}
	return info.State.Running
}

// GetClusterMetrics returns cluster metrics
func (s *nginxClusterService) GetClusterMetrics(ctx context.Context, clusterID string) (*dto.NginxClusterMetricsResponse, error) {
	nodes, err := s.clusterRepo.ListNodes(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	nodeMetrics := make([]dto.NginxNodeMetrics, 0, len(nodes))
	var totalRequests int64
	var totalActiveConns int

	for _, node := range nodes {
		// Get nginx stub_status metrics
		metrics := s.getNginxMetrics(ctx, &node)
		nodeMetrics = append(nodeMetrics, metrics)
		totalRequests += metrics.TotalRequests
		totalActiveConns += metrics.ActiveConns
	}

	return &dto.NginxClusterMetricsResponse{
		ClusterID:     clusterID,
		TotalRequests: totalRequests,
		ActiveConns:   totalActiveConns,
		NodeMetrics:   nodeMetrics,
	}, nil
}

// getNginxMetrics gets metrics from a single node
func (s *nginxClusterService) getNginxMetrics(ctx context.Context, node *entities.NginxNode) dto.NginxNodeMetrics {
	return dto.NginxNodeMetrics{
		NodeID:      node.ID,
		NodeName:    node.Name,
		Role:        node.Role,
		ActiveConns: 0,
	}
}

// TestConnection tests connection to the cluster
func (s *nginxClusterService) TestConnection(ctx context.Context, clusterID string) (*dto.TestNginxConnectionResponse, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	masterNode, err := s.clusterRepo.FindNodeByID(cluster.MasterNodeID)
	if err != nil {
		return nil, fmt.Errorf("master node not found: %w", err)
	}

	// Test HTTP connection
	url := fmt.Sprintf("http://localhost:%d/health", masterNode.HTTPPort)
	start := time.Now()
	resp, err := http.Get(url)
	latency := time.Since(start)

	if err != nil {
		return &dto.TestNginxConnectionResponse{
			Success:  false,
			Message:  fmt.Sprintf("Connection failed: %v", err),
			NodeName: masterNode.Name,
			NodeRole: masterNode.Role,
		}, nil
	}
	defer resp.Body.Close()

	return &dto.TestNginxConnectionResponse{
		Success:    true,
		Message:    "Connection successful",
		Latency:    latency.String(),
		NodeName:   masterNode.Name,
		NodeRole:   masterNode.Role,
		StatusCode: resp.StatusCode,
	}, nil
}

// GetConnectionInfo returns connection information
func (s *nginxClusterService) GetConnectionInfo(ctx context.Context, clusterID string) (*dto.NginxConnectionInfoResponse, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	infra, _ := s.infraRepo.FindByID(cluster.InfrastructureID)
	nodes, _ := s.clusterRepo.ListNodes(clusterID)

	var masterEndpoint dto.NodeEndpoint
	backupEndpoints := make([]dto.NodeEndpoint, 0)

	for _, node := range nodes {
		endpoint := dto.NodeEndpoint{
			NodeID:    node.ID,
			NodeName:  node.Name,
			Role:      node.Role,
			IP:        "localhost",
			HTTPPort:  node.HTTPPort,
			HTTPSPort: node.HTTPSPort,
			HTTPURL:   fmt.Sprintf("http://localhost:%d", node.HTTPPort),
			IsHealthy: node.IsHealthy,
		}
		if node.HTTPSPort > 0 {
			endpoint.HTTPSURL = fmt.Sprintf("https://localhost:%d", node.HTTPSPort)
		}

		if node.ID == cluster.MasterNodeID {
			masterEndpoint = endpoint
		} else {
			backupEndpoints = append(backupEndpoints, endpoint)
		}
	}

	// Get server names
	serverBlocks, _ := s.clusterRepo.ListServerBlocks(clusterID)
	serverNames := make([]string, 0)
	for _, block := range serverBlocks {
		serverNames = append(serverNames, block.ServerName)
	}

	endpoints := dto.NginxConnectionEndpoints{
		MasterNode:  masterEndpoint,
		BackupNodes: backupEndpoints,
	}

	if cluster.VirtualIP != "" {
		endpoints.VirtualIP = &dto.VIPEndpoint{
			IP:       cluster.VirtualIP,
			HTTPPort: cluster.HTTPPort,
			HTTPURL:  fmt.Sprintf("http://%s:%d", cluster.VirtualIP, cluster.HTTPPort),
		}
		if cluster.HTTPSPort > 0 {
			endpoints.VirtualIP.HTTPSPort = cluster.HTTPSPort
			endpoints.VirtualIP.HTTPSURL = fmt.Sprintf("https://%s:%d", cluster.VirtualIP, cluster.HTTPSPort)
		}
	}

	return &dto.NginxConnectionInfoResponse{
		ClusterID:   clusterID,
		ClusterName: cluster.ClusterName,
		Status:      string(infra.Status),
		Endpoints:   endpoints,
		ServerNames: serverNames,
	}, nil
}

// TriggerFailover manually triggers failover to a specified node
func (s *nginxClusterService) TriggerFailover(ctx context.Context, clusterID string, req dto.TriggerNginxFailoverRequest) (*dto.NginxFailoverResponse, error) {
	cluster, err := s.clusterRepo.FindByID(clusterID)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}

	targetNode, err := s.clusterRepo.FindNodeByID(req.TargetNodeID)
	if err != nil {
		return nil, fmt.Errorf("target node not found: %w", err)
	}

	oldMaster, _ := s.clusterRepo.FindNodeByID(cluster.MasterNodeID)

	start := time.Now()

	// Demote old master
	if oldMaster != nil {
		oldMaster.Role = "backup"
		oldMaster.Priority = 50
		s.clusterRepo.UpdateNode(oldMaster)
	}

	// Promote new master
	targetNode.Role = "master"
	targetNode.Priority = 100
	s.clusterRepo.UpdateNode(targetNode)

	cluster.MasterNodeID = targetNode.ID
	s.clusterRepo.Update(cluster)

	// Record failover event
	event := &entities.NginxFailoverEvent{
		ID:            uuid.New().String(),
		ClusterID:     clusterID,
		OldMasterID:   oldMaster.ID,
		OldMasterName: oldMaster.Name,
		NewMasterID:   targetNode.ID,
		NewMasterName: targetNode.Name,
		Reason:        req.Reason,
		TriggeredBy:   "user",
	}
	s.clusterRepo.CreateFailoverEvent(event)

	duration := time.Since(start)

	return &dto.NginxFailoverResponse{
		Success:       true,
		Message:       "Failover completed successfully",
		OldMasterID:   oldMaster.ID,
		OldMasterName: oldMaster.Name,
		NewMasterID:   targetNode.ID,
		NewMasterName: targetNode.Name,
		Duration:      duration.String(),
	}, nil
}

// GetFailoverHistory returns failover history
func (s *nginxClusterService) GetFailoverHistory(ctx context.Context, clusterID string) (*dto.NginxFailoverHistoryResponse, error) {
	events, err := s.clusterRepo.ListFailoverEvents(clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to list failover events: %w", err)
	}

	eventDTOs := make([]dto.NginxFailoverEvent, 0, len(events))
	for _, e := range events {
		eventDTOs = append(eventDTOs, dto.NginxFailoverEvent{
			ID:            e.ID,
			OldMasterID:   e.OldMasterID,
			OldMasterName: e.OldMasterName,
			NewMasterID:   e.NewMasterID,
			NewMasterName: e.NewMasterName,
			Reason:        e.Reason,
			TriggeredBy:   e.TriggeredBy,
			OccurredAt:    e.OccurredAt.Format(time.RFC3339),
		})
	}

	return &dto.NginxFailoverHistoryResponse{
		ClusterID: clusterID,
		Events:    eventDTOs,
	}, nil
}

// helper func() to update infrastructure status
func (s *nginxClusterService) updateInfraStatus(infraID string, status entities.InfrastructureStatus) {
	infra, err := s.infraRepo.FindByID(infraID)
	if err == nil {
		infra.Status = status
		s.infraRepo.Update(infra)
	}
}


// helper func() to cleanup resources
func (s *nginxClusterService) cleanup(ctx context.Context, cluster *entities.NginxCluster, networkID string) {
	nodes, _ := s.clusterRepo.ListNodes(cluster.ID)
	for _, node := range nodes {
		if node.ContainerID != "" {
			s.dockerSvc.StopContainer(ctx, node.ContainerID)
			s.dockerSvc.RemoveContainer(ctx, node.ContainerID)
		}
		s.clusterRepo.DeleteNode(node.ID)
	}
	if networkID != "" {
		s.dockerSvc.RemoveNetwork(ctx, networkID)
	}
	s.clusterRepo.Delete(cluster.ID)
	s.infraRepo.Delete(cluster.InfrastructureID)
}


// helper func() to publish events to Kafka
func (s *nginxClusterService) publishEvent(ctx context.Context, eventType, infraID, clusterID, status string) {
	if s.kafkaProducer != nil {
		event := kafka.InfrastructureEvent{
			InstanceID: clusterID,
			Type:       "nginx_cluster",
			Action:     eventType,
			Timestamp:  time.Now(),
			Metadata: map[string]interface{}{
				"infra_id": infraID,
				"status":   status,
			},
		}
		s.kafkaProducer.PublishEvent(ctx, event)
	}
}

// helper func() to fincd suitable port
func (s *nginxClusterService) getNextAvailablePortForCluster(cluster *entities.NginxCluster) (int, int) {
	nodes, _ := s.clusterRepo.ListNodes(cluster.ID)
	usedPorts := make(map[int]bool)
	for _, node := range nodes {
		usedPorts[node.HTTPPort] = true
		if node.HTTPSPort > 0 {
			usedPorts[node.HTTPSPort] = true
		}
	}

	httpPort := cluster.HTTPPort
	for i := 0; i < 100; i++ {
		candidatePort := cluster.HTTPPort + i
		if !usedPorts[candidatePort] && s.isPortAvailable(candidatePort) {
			httpPort = candidatePort
			break
		}
	}

	httpsPort := 0
	if cluster.HTTPSPort > 0 {
		for i := 0; i < 100; i++ {
			candidatePort := cluster.HTTPSPort + i
			if !usedPorts[candidatePort] && s.isPortAvailable(candidatePort) {
				httpsPort = candidatePort
				break
			}
		}
	}

	return httpPort, httpsPort
}

func (s *nginxClusterService) isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}
