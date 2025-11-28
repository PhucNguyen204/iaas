package http

import (
	"net/http"
	"strconv"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DinDHandler struct {
	dinDService services.IDinDService
	logger      logger.ILogger
}

func NewDinDHandler(dinDService services.IDinDService, logger logger.ILogger) *DinDHandler {
	return &DinDHandler{
		dinDService: dinDService,
		logger:      logger,
	}
}

func (h *DinDHandler) RegisterRoutes(r *gin.RouterGroup) {
	dind := r.Group("/dind")
	{
		// Environment management
		dind.POST("/environments", h.CreateEnvironment)
		dind.GET("/environments", h.ListEnvironments)
		dind.GET("/environments/:id", h.GetEnvironment)
		dind.DELETE("/environments/:id", h.DeleteEnvironment)
		dind.POST("/environments/:id/start", h.StartEnvironment)
		dind.POST("/environments/:id/stop", h.StopEnvironment)

		// Docker operations inside DinD
		dind.POST("/environments/:id/exec", h.ExecCommand)
		dind.POST("/environments/:id/build", h.BuildImage)
		dind.POST("/environments/:id/compose", h.RunCompose)
		dind.POST("/environments/:id/pull", h.PullImage)

		// Info retrieval
		dind.GET("/environments/:id/containers", h.ListContainers)
		dind.GET("/environments/:id/images", h.ListImages)
		dind.GET("/environments/:id/logs", h.GetLogs)
		dind.GET("/environments/:id/stats", h.GetStats)
	}
}

// CreateEnvironment creates a new DinD environment
// @Summary Create DinD Environment
// @Description Create a new Docker-in-Docker isolated environment
// @Tags DinD
// @Accept json
// @Produce json
// @Param request body dto.CreateDinDEnvironmentRequest true "Environment configuration"
// @Success 201 {object} dto.APIResponse
// @Failure 400 {object} dto.APIResponse
// @Failure 500 {object} dto.APIResponse
// @Router /dind/environments [post]
func (h *DinDHandler) CreateEnvironment(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{
			Success: false,
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	var req dto.CreateDinDEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	h.logger.Info("creating DinD environment",
		zap.String("name", req.Name),
		zap.String("user_id", userID))

	env, err := h.dinDService.CreateEnvironment(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("failed to create DinD environment", zap.Error(err))
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create DinD environment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "DinD environment created successfully",
		Data:    env,
	})
}

// ListEnvironments lists all DinD environments for the user
func (h *DinDHandler) ListEnvironments(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, dto.APIResponse{
			Success: false,
			Code:    "UNAUTHORIZED",
			Message: "User not authenticated",
		})
		return
	}

	envs, err := h.dinDService.ListEnvironments(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to list environments",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Environments retrieved successfully",
		Data:    envs,
	})
}

// GetEnvironment gets info about a specific DinD environment
// Supports both environment ID and infrastructure ID
func (h *DinDHandler) GetEnvironment(c *gin.Context) {
	id := c.Param("id")

	// First try to get by environment ID
	env, err := h.dinDService.GetEnvironment(c.Request.Context(), id)
	if err != nil {
		// Fallback: try to get by infrastructure ID
		env, err = h.dinDService.GetEnvironmentByInfraID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, dto.APIResponse{
				Success: false,
				Code:    "NOT_FOUND",
				Message: "Environment not found",
				Error:   err.Error(),
			})
			return
		}
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Environment retrieved successfully",
		Data:    env,
	})
}

// DeleteEnvironment deletes a DinD environment
func (h *DinDHandler) DeleteEnvironment(c *gin.Context) {
	id := c.Param("id")

	if err := h.dinDService.DeleteEnvironment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to delete environment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Environment deleted successfully",
	})
}

// StartEnvironment starts a stopped DinD environment
func (h *DinDHandler) StartEnvironment(c *gin.Context) {
	id := c.Param("id")

	if err := h.dinDService.StartEnvironment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to start environment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Environment started successfully",
	})
}

// StopEnvironment stops a running DinD environment
func (h *DinDHandler) StopEnvironment(c *gin.Context) {
	id := c.Param("id")

	if err := h.dinDService.StopEnvironment(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to stop environment",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Environment stopped successfully",
	})
}

// ExecCommand executes a docker command inside the DinD environment
// @Summary Execute Docker Command
// @Description Run any docker command inside the DinD environment
// @Tags DinD
// @Accept json
// @Produce json
// @Param id path string true "Environment ID"
// @Param request body dto.ExecCommandRequest true "Command to execute"
// @Success 200 {object} dto.APIResponse
// @Router /dind/environments/{id}/exec [post]
func (h *DinDHandler) ExecCommand(c *gin.Context) {
	id := c.Param("id")

	var req dto.ExecCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	h.logger.Info("executing command in DinD",
		zap.String("env_id", id),
		zap.String("command", req.Command))

	resp, err := h.dinDService.ExecCommand(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to execute command",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Command executed successfully",
		Data:    resp,
	})
}

// BuildImage builds a Docker image inside DinD environment
func (h *DinDHandler) BuildImage(c *gin.Context) {
	id := c.Param("id")

	var req dto.BuildImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	h.logger.Info("building image in DinD",
		zap.String("env_id", id),
		zap.String("image_name", req.ImageName))

	resp, err := h.dinDService.BuildImage(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to build image",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Image built successfully",
		Data:    resp,
	})
}

// RunCompose runs docker-compose inside DinD environment
func (h *DinDHandler) RunCompose(c *gin.Context) {
	id := c.Param("id")

	var req dto.ComposeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	h.logger.Info("running docker-compose in DinD",
		zap.String("env_id", id),
		zap.String("action", req.Action))

	resp, err := h.dinDService.RunCompose(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to run docker-compose",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Docker-compose executed successfully",
		Data:    resp,
	})
}

// PullImage pulls a Docker image inside DinD environment
func (h *DinDHandler) PullImage(c *gin.Context) {
	id := c.Param("id")

	var req dto.PullImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.APIResponse{
			Success: false,
			Code:    "INVALID_REQUEST",
			Message: "Invalid request body",
			Error:   err.Error(),
		})
		return
	}

	resp, err := h.dinDService.PullImage(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to pull image",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Image pulled successfully",
		Data:    resp,
	})
}

// ListContainers lists all containers inside DinD environment
func (h *DinDHandler) ListContainers(c *gin.Context) {
	id := c.Param("id")

	resp, err := h.dinDService.ListContainers(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to list containers",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Containers retrieved successfully",
		Data:    resp,
	})
}

// ListImages lists all images inside DinD environment
func (h *DinDHandler) ListImages(c *gin.Context) {
	id := c.Param("id")

	resp, err := h.dinDService.ListImages(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to list images",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Images retrieved successfully",
		Data:    resp,
	})
}

// GetLogs gets logs from DinD container
func (h *DinDHandler) GetLogs(c *gin.Context) {
	id := c.Param("id")
	tailStr := c.DefaultQuery("tail", "100")
	tail, _ := strconv.Atoi(tailStr)

	resp, err := h.dinDService.GetLogs(c.Request.Context(), id, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to get logs",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Logs retrieved successfully",
		Data:    resp,
	})
}

// GetStats gets resource statistics of DinD environment
func (h *DinDHandler) GetStats(c *gin.Context) {
	id := c.Param("id")

	resp, err := h.dinDService.GetStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.APIResponse{
			Success: false,
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to get stats",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.APIResponse{
		Success: true,
		Code:    "SUCCESS",
		Message: "Stats retrieved successfully",
		Data:    resp,
	})
}
