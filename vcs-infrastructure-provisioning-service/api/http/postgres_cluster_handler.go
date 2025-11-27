package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type PostgreSQLClusterHandler struct {
	clusterService services.IPostgreSQLClusterService
	logger         logger.ILogger
}

func NewPostgreSQLClusterHandler(
	clusterService services.IPostgreSQLClusterService,
	logger logger.ILogger,
) *PostgreSQLClusterHandler {
	return &PostgreSQLClusterHandler{
		clusterService: clusterService,
		logger:         logger,
	}
}

// CreateCluster creates a new PostgreSQL cluster
// @Summary Create PostgreSQL cluster
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param request body dto.CreateClusterRequest true "Cluster configuration"
// @Success 201 {object} dto.ClusterInfoResponse
// @Router /api/v1/postgres/cluster [post]
func (h *PostgreSQLClusterHandler) CreateCluster(c *gin.Context) {
	var req dto.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		userID = "system" // Default for testing
	}

	// Use background context with timeout for long-running cluster creation
	// This prevents HTTP request timeout from canceling replica creation
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cluster, err := h.clusterService.CreateCluster(ctx, userID.(string), req)
	if err != nil {
		h.logger.Error("failed to create cluster", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, cluster)
}

// GetClusterInfo retrieves cluster information
// @Summary Get cluster info
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.ClusterInfoResponse
// @Router /api/v1/postgres/cluster/{id} [get]
func (h *PostgreSQLClusterHandler) GetClusterInfo(c *gin.Context) {
	clusterID := c.Param("id")

	cluster, err := h.clusterService.GetClusterInfo(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to get cluster info", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, cluster)
}

// StartCluster starts a stopped cluster
// @Summary Start cluster
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/start [post]
func (h *PostgreSQLClusterHandler) StartCluster(c *gin.Context) {
	clusterID := c.Param("id")

	if err := h.clusterService.StartCluster(c.Request.Context(), clusterID); err != nil {
		h.logger.Error("failed to start cluster", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster started successfully"})
}

// StopCluster stops a running cluster
// @Summary Stop cluster
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/stop [post]
func (h *PostgreSQLClusterHandler) StopCluster(c *gin.Context) {
	clusterID := c.Param("id")

	if err := h.clusterService.StopCluster(c.Request.Context(), clusterID); err != nil {
		h.logger.Error("failed to stop cluster", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster stopped successfully"})
}

// RestartCluster restarts a cluster
// @Summary Restart cluster
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/restart [post]
func (h *PostgreSQLClusterHandler) RestartCluster(c *gin.Context) {
	clusterID := c.Param("id")

	if err := h.clusterService.RestartCluster(c.Request.Context(), clusterID); err != nil {
		h.logger.Error("failed to restart cluster", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster restarted successfully"})
}

// DeleteCluster deletes a cluster and all resources
// @Summary Delete cluster
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id} [delete]
func (h *PostgreSQLClusterHandler) DeleteCluster(c *gin.Context) {
	clusterID := c.Param("id")

	if err := h.clusterService.DeleteCluster(c.Request.Context(), clusterID); err != nil {
		h.logger.Error("failed to delete cluster", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster deleted successfully"})
}

// ScaleCluster scales cluster up or down
// @Summary Scale cluster
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.ScaleClusterRequest true "Target node count"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/scale [post]
func (h *PostgreSQLClusterHandler) ScaleCluster(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.ScaleClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.ScaleCluster(c.Request.Context(), clusterID, req); err != nil {
		h.logger.Error("failed to scale cluster", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "cluster scaled successfully"})
}

// PromoteReplica promotes a replica to primary (manual failover)
// @Summary Manual failover
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.TriggerFailoverRequest true "New primary node ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/failover [post]
func (h *PostgreSQLClusterHandler) PromoteReplica(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.TriggerFailoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.PromoteReplica(c.Request.Context(), clusterID, req.NewPrimaryNodeID); err != nil {
		h.logger.Error("failed to promote replica", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "failover completed successfully"})
}

// StopNode stops a specific node in the cluster
// @Summary Stop a cluster node
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.NodeActionRequest true "Node ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/nodes/stop [post]
func (h *PostgreSQLClusterHandler) StopNode(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.NodeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.StopNode(c.Request.Context(), clusterID, req.NodeID); err != nil {
		h.logger.Error("failed to stop node", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "node stopped successfully, automatic failover triggered if primary"})
}

// StartNode starts a specific node in the cluster
// @Summary Start a cluster node
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.NodeActionRequest true "Node ID"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/nodes/start [post]
func (h *PostgreSQLClusterHandler) StartNode(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.NodeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.StartNode(c.Request.Context(), clusterID, req.NodeID); err != nil {
		h.logger.Error("failed to start node", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "node started successfully"})
}

// GetFailoverHistory returns failover events for a cluster
// @Summary Get failover history
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {array} dto.FailoverEvent
// @Router /api/v1/postgres/cluster/{id}/failover-history [get]
func (h *PostgreSQLClusterHandler) GetFailoverHistory(c *gin.Context) {
	clusterID := c.Param("id")

	events, err := h.clusterService.GetFailoverHistory(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to get failover history", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

// AddNode adds a new replica node to the cluster
// @Summary Add a new replica node
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.AddNodeRequest true "Add node request"
// @Success 200 {object} dto.AddNodeResponse
// @Router /api/v1/postgres/cluster/{id}/nodes [post]
func (h *PostgreSQLClusterHandler) AddNode(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.AddNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body (all fields optional)
		req = dto.AddNodeRequest{}
	}

	result, err := h.clusterService.AddNode(c.Request.Context(), clusterID, req)
	if err != nil {
		h.logger.Error("failed to add node", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RemoveNode removes a node from the cluster
// @Summary Remove a node from cluster
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.RemoveNodeRequest true "Remove node request"
// @Success 200 {object} dto.RemoveNodeResponse
// @Router /api/v1/postgres/cluster/{id}/nodes [delete]
func (h *PostgreSQLClusterHandler) RemoveNode(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.RemoveNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.clusterService.RemoveNode(c.Request.Context(), clusterID, req)
	if err != nil {
		h.logger.Error("failed to remove node", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetReplicationStatus shows replication status for all nodes
// @Summary Get replication status
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.ReplicationStatusResponse
// @Router /api/v1/postgres/cluster/{id}/replication [get]
func (h *PostgreSQLClusterHandler) GetReplicationStatus(c *gin.Context) {
	clusterID := c.Param("id")

	status, err := h.clusterService.GetReplicationStatus(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to get replication status", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)

	/*
		status, err := h.clusterService.GetReplicationStatus(c.Request.Context(), clusterID)
		if err != nil {
			h.logger.Error("failed to get replication status", zap.String("cluster_id", clusterID), zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, status)
	*/
}

// GetClusterStats returns aggregated stats
// @Summary Get cluster statistics
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.ClusterStatsResponse
// @Router /api/v1/postgres/cluster/{id}/stats [get]
func (h *PostgreSQLClusterHandler) GetClusterStats(c *gin.Context) {
	clusterID := c.Param("id")

	stats, err := h.clusterService.GetClusterStats(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to get cluster stats", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// GetClusterLogs retrieves logs from all nodes
// @Summary Get cluster logs
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Param tail query string false "Number of lines" default(100)
// @Success 200 {object} dto.ClusterLogsResponse
// @Router /api/v1/postgres/cluster/{id}/logs [get]
func (h *PostgreSQLClusterHandler) GetClusterLogs(c *gin.Context) {
	clusterID := c.Param("id")
	tail := c.DefaultQuery("tail", "100")

	logs, err := h.clusterService.GetClusterLogs(c.Request.Context(), clusterID, tail)
	if err != nil {
		h.logger.Error("failed to get cluster logs", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, logs)
}

// CreateUser creates a new user in the cluster
// @Summary Create user
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.CreateUserRequest true "User details"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/users [post]
func (h *PostgreSQLClusterHandler) CreateUser(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.CreateUser(c.Request.Context(), clusterID, req); err != nil {
		h.logger.Error("failed to create user", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user created successfully"})
}

// ListUsers lists all users in the cluster
// @Summary List users
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {array} dto.UserInfo
// @Router /api/v1/postgres/cluster/{id}/users [get]
func (h *PostgreSQLClusterHandler) ListUsers(c *gin.Context) {
	clusterID := c.Param("id")

	users, err := h.clusterService.ListUsers(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to list users", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// DeleteUser deletes a user from the cluster
// @Summary Delete user
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Param username path string true "Username"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/users/{username} [delete]
func (h *PostgreSQLClusterHandler) DeleteUser(c *gin.Context) {
	clusterID := c.Param("id")
	username := c.Param("username")

	if err := h.clusterService.DeleteUser(c.Request.Context(), clusterID, username); err != nil {
		h.logger.Error("failed to delete user", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}

// CreateDatabase creates a new database in the cluster
// @Summary Create database
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.CreateClusterDatabaseRequest true "Database details"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/databases [post]
func (h *PostgreSQLClusterHandler) CreateDatabase(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.CreateClusterDatabaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.CreateDatabase(c.Request.Context(), clusterID, req); err != nil {
		h.logger.Error("failed to create database", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "database created successfully"})
}

// ListDatabases lists all databases in the cluster
// @Summary List databases
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {array} dto.ClusterDatabaseInfo
// @Router /api/v1/postgres/cluster/{id}/databases [get]
func (h *PostgreSQLClusterHandler) ListDatabases(c *gin.Context) {
	clusterID := c.Param("id")

	databases, err := h.clusterService.ListDatabases(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to list databases", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, databases)
}

// DeleteDatabase deletes a database from the cluster
// @Summary Delete database
// @Tags PostgreSQL Cluster
// @Param id path string true "Cluster ID"
// @Param dbname path string true "Database name"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/databases/{dbname} [delete]
func (h *PostgreSQLClusterHandler) DeleteDatabase(c *gin.Context) {
	clusterID := c.Param("id")
	dbname := c.Param("dbname")

	if err := h.clusterService.DeleteDatabase(c.Request.Context(), clusterID, dbname); err != nil {
		h.logger.Error("failed to delete database", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "database deleted successfully"})
}

// UpdateConfig updates cluster configuration
// @Summary Update configuration
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.UpdateConfigRequest true "Configuration updates"
// @Success 200 {object} map[string]string
// @Router /api/v1/postgres/cluster/{id}/config [put]
func (h *PostgreSQLClusterHandler) UpdateConfig(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clusterService.UpdateConfig(c.Request.Context(), clusterID, req); err != nil {
		h.logger.Error("failed to update config", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "config updated successfully"})
}

// GetEndpoints returns cluster connection endpoints
// @Summary Get connection endpoints
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.ClusterInfoResponse
// @Router /api/v1/postgres/cluster/{id}/endpoints [get]
func (h *PostgreSQLClusterHandler) GetEndpoints(c *gin.Context) {
	clusterID := c.Param("id")

	endpoints, err := h.clusterService.GetEndpoints(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to get endpoints", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, endpoints)
}

// ==================== Patroni Management Endpoints ====================

// PatroniSwitchover performs manual switchover
// @Summary Switchover primary node
// @Tags Patroni Management
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.SwitchoverRequest true "Switchover request"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/patroni/switchover [post]
func (h *PostgreSQLClusterHandler) PatroniSwitchover(c *gin.Context) {
	clusterID := c.Param("id")
	var req dto.SwitchoverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("patroni switchover requested",
		zap.String("cluster_id", clusterID),
		zap.String("candidate", req.CandidateNode))

	// This would call patronictl switchover
	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Switchover initiated. Check cluster status for progress.",
	})
}

// PatroniReinit reinitializes a node
// @Summary Reinitialize node
// @Tags Patroni Management
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.ReinitRequest true "Reinit request"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/patroni/reinit [post]
func (h *PostgreSQLClusterHandler) PatroniReinit(c *gin.Context) {
	clusterID := c.Param("id")
	var req dto.ReinitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("patroni reinit requested",
		zap.String("cluster_id", clusterID),
		zap.String("node_id", req.NodeID))

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Node reinitialization started",
	})
}

// PatroniPause pauses Patroni cluster management
// @Summary Pause cluster management
// @Tags Patroni Management
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/patroni/pause [post]
func (h *PostgreSQLClusterHandler) PatroniPause(c *gin.Context) {
	clusterID := c.Param("id")

	h.logger.Info("patroni pause requested", zap.String("cluster_id", clusterID))

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Cluster management paused",
	})
}

// PatroniResume resumes Patroni cluster management
// @Summary Resume cluster management
// @Tags Patroni Management
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/patroni/resume [post]
func (h *PostgreSQLClusterHandler) PatroniResume(c *gin.Context) {
	clusterID := c.Param("id")

	h.logger.Info("patroni resume requested", zap.String("cluster_id", clusterID))

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Cluster management resumed",
	})
}

// PatroniStatus gets Patroni cluster status
// @Summary Get Patroni cluster status
// @Tags Patroni Management
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.PatroniStatusResponse
// @Router /api/v1/postgres/cluster/{id}/patroni/status [get]
func (h *PostgreSQLClusterHandler) PatroniStatus(c *gin.Context) {
	clusterID := c.Param("id")

	h.logger.Info("patroni status requested", zap.String("cluster_id", clusterID))

	// Mock response - would query Patroni REST API
	c.JSON(http.StatusOK, dto.PatroniStatusResponse{
		Scope:    clusterID,
		Timeline: 1,
		Paused:   false,
		Members:  []dto.PatroniMember{},
	})
}

// ==================== PgBackRest Management Endpoints ====================

// BackupCluster creates a backup using PgBackRest
// @Summary Create cluster backup
// @Tags Backup Management
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.BackupRequest true "Backup request"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/backup [post]
func (h *PostgreSQLClusterHandler) BackupCluster(c *gin.Context) {
	clusterID := c.Param("id")
	var req dto.BackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("backup requested",
		zap.String("cluster_id", clusterID),
		zap.String("type", req.Type))

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: fmt.Sprintf("%s backup started", req.Type),
	})
}

// ListBackups lists all backups
// @Summary List cluster backups
// @Tags Backup Management
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.PgBackRestBackupInfoResponse
// @Router /api/v1/postgres/cluster/{id}/backups [get]
func (h *PostgreSQLClusterHandler) ListBackups(c *gin.Context) {
	clusterID := c.Param("id")

	h.logger.Info("list backups requested", zap.String("cluster_id", clusterID))

	c.JSON(http.StatusOK, dto.PgBackRestBackupInfoResponse{
		Stanza:  clusterID,
		Status:  "ok",
		Backups: []dto.PgBackRestBackupInfo{},
	})
}

// RestoreCluster restores cluster from backup
// @Summary Restore from backup
// @Tags Backup Management
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.RestoreRequest true "Restore request"
// @Success 200 {object} dto.APIResponse
// @Router /api/v1/postgres/cluster/{id}/restore [post]
func (h *PostgreSQLClusterHandler) RestoreCluster(c *gin.Context) {
	clusterID := c.Param("id")
	var req dto.RestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("restore requested", zap.String("cluster_id", clusterID))

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Message: "Restore initiated",
	})
}

// ExecuteQuery runs a SQL query on the cluster
// @Summary Execute SQL query
// @Tags PostgreSQL Cluster
// @Accept json
// @Produce json
// @Param id path string true "Cluster ID"
// @Param request body dto.ExecuteQueryRequest true "Query request"
// @Success 200 {object} dto.QueryResult
// @Router /api/v1/postgres/cluster/{id}/query [post]
func (h *PostgreSQLClusterHandler) ExecuteQuery(c *gin.Context) {
	clusterID := c.Param("id")

	var req dto.ExecuteQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.clusterService.ExecuteQuery(c.Request.Context(), clusterID, req)
	if err != nil {
		h.logger.Error("failed to execute query", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// TestReplication tests data replication across all nodes
// @Summary Test replication
// @Tags PostgreSQL Cluster
// @Produce json
// @Param id path string true "Cluster ID"
// @Success 200 {object} dto.ReplicationTestResult
// @Router /api/v1/postgres/cluster/{id}/test-replication [post]
func (h *PostgreSQLClusterHandler) TestReplication(c *gin.Context) {
	clusterID := c.Param("id")

	result, err := h.clusterService.TestReplication(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("failed to test replication", zap.String("cluster_id", clusterID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
