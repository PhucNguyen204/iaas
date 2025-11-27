package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/kafka"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IPostgreSQLService interface {
	CreatePostgreSQL(ctx context.Context, userID string, req dto.CreatePostgreSQLRequest) (*dto.PostgreSQLInfoResponse, error)
	StartPostgreSQL(ctx context.Context, id string) error
	StopPostgreSQL(ctx context.Context, id string) error
	RestartPostgreSQL(ctx context.Context, id string) error
	DeletePostgreSQL(ctx context.Context, id string) error
	GetPostgreSQLInfo(ctx context.Context, id string) (*dto.PostgreSQLInfoResponse, error)
	GetPostgreSQLLogs(ctx context.Context, id string, tail string) (string, error)
	GetPostgreSQLStats(ctx context.Context, id string) (*dto.PostgreSQLStatsResponse, error)
	BackupPostgreSQL(ctx context.Context, id string, req dto.BackupPostgreSQLRequest) (*dto.BackupPostgreSQLResponse, error)
	RestorePostgreSQL(ctx context.Context, id string, req dto.RestorePostgreSQLRequest) error
}

type postgreSQLService struct {
	infraRepo     repositories.IInfrastructureRepository
	pgRepo        repositories.IPostgreSQLRepository
	dockerSvc     docker.IDockerService
	kafkaProducer kafka.IKafkaProducer
	logger        logger.ILogger
}

func NewPostgreSQLService(
	infraRepo repositories.IInfrastructureRepository,
	pgRepo repositories.IPostgreSQLRepository,
	dockerSvc docker.IDockerService,
	kafkaProducer kafka.IKafkaProducer,
	logger logger.ILogger,
) IPostgreSQLService {
	return &postgreSQLService{
		infraRepo:     infraRepo,
		pgRepo:        pgRepo,
		dockerSvc:     dockerSvc,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

func (s *postgreSQLService) CreatePostgreSQL(ctx context.Context, userID string, req dto.CreatePostgreSQLRequest) (*dto.PostgreSQLInfoResponse, error) {
	infraID := uuid.New().String()
	instanceID := uuid.New().String()

	infra := &entities.Infrastructure{
		ID:     infraID,
		Name:   req.Name,
		Type:   entities.TypePostgreSQLSingle,
		Status: entities.StatusCreating,
		UserID: userID,
	}

	if err := s.infraRepo.Create(infra); err != nil {
		s.logger.Error("failed to create infrastructure record", zap.Error(err))
		return nil, err
	}

	instance := &entities.PostgreSQLInstance{
		ID:               instanceID,
		InfrastructureID: infraID,
		Version:          req.Version,
		Port:             req.Port,
		DatabaseName:     req.DatabaseName,
		Username:         req.Username,
		Password:         req.Password,
		CPULimit:         req.CPULimit,
		MemoryLimit:      req.MemoryLimit,
		StorageSize:      req.StorageSize,
	}

	if err := s.pgRepo.Create(instance); err != nil {
		s.logger.Error("failed to create postgres instance record", zap.Error(err))
		return nil, err
	}

	volumeName := fmt.Sprintf("iaas-postgres-%s", instanceID)
	if err := s.dockerSvc.CreateVolume(ctx, volumeName); err != nil {
		s.logger.Error("failed to create volume", zap.Error(err))
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		return nil, err
	}

	instance.VolumeID = volumeName

	containerName := fmt.Sprintf("iaas-postgres-%s", instanceID)
	containerConfig := docker.ContainerConfig{
		Name:  containerName,
		Image: fmt.Sprintf("postgres:%s", req.Version),
		Env: []string{
			fmt.Sprintf("POSTGRES_USER=%s", req.Username),
			fmt.Sprintf("POSTGRES_PASSWORD=%s", req.Password),
			fmt.Sprintf("POSTGRES_DB=%s", req.DatabaseName),
		},
		Ports: map[string]string{
			"5432": fmt.Sprintf("%d", req.Port),
		},
		Volumes: map[string]string{
			volumeName: "/var/lib/postgresql/data",
		},
		Network: "iaas_iaas-network",
		Resources: docker.ResourceConfig{
			CPULimit:    req.CPULimit,
			MemoryLimit: req.MemoryLimit,
		},
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, containerConfig)
	if err != nil {
		s.logger.Error("failed to create container", zap.Error(err))
		s.dockerSvc.RemoveVolume(ctx, volumeName)
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		return nil, err
	}

	instance.ContainerID = containerID

	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		s.logger.Error("failed to start container", zap.Error(err))
		s.dockerSvc.RemoveContainer(ctx, containerID)
		s.dockerSvc.RemoveVolume(ctx, volumeName)
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		return nil, err
	}

	infra.Status = entities.StatusRunning
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	if err := s.pgRepo.Update(instance); err != nil {
		s.logger.Error("failed to update postgres instance", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: infraID,
		UserID:     userID,
		Type:       "postgres_single",
		Action:     "created",
		Metadata: map[string]interface{}{
			"name":    req.Name,
			"version": req.Version,
			"port":    req.Port,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	s.logger.Info("postgresql created successfully",
		zap.String("instance_id", instanceID),
		zap.String("container_id", containerID))

	return &dto.PostgreSQLInfoResponse{
		ID:           infraID,
		Name:         infra.Name,
		Status:       string(infra.Status),
		ContainerID:  containerID,
		Version:      req.Version,
		Port:         req.Port,
		DatabaseName: req.DatabaseName,
		Username:     req.Username,
		CPULimit:     req.CPULimit,
		MemoryLimit:  req.MemoryLimit,
		StorageSize:  req.StorageSize,
		CreatedAt:    infra.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    infra.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *postgreSQLService) StartPostgreSQL(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return err
	}

	if err := s.dockerSvc.StartContainer(ctx, instance.ContainerID); err != nil {
		s.logger.Error("failed to start container", zap.Error(err))
		return err
	}

	infra.Status = entities.StatusRunning
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     infra.UserID,
		Type:       "postgres_single",
		Action:     "started",
		Metadata: map[string]interface{}{
			"container_id": instance.ContainerID,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *postgreSQLService) StopPostgreSQL(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return err
	}

	if err := s.dockerSvc.StopContainer(ctx, instance.ContainerID); err != nil {
		s.logger.Error("failed to stop container", zap.Error(err))
		return err
	}

	infra.Status = entities.StatusStopped
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     infra.UserID,
		Type:       "postgres_single",
		Action:     "stopped",
		Metadata: map[string]interface{}{
			"container_id": instance.ContainerID,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *postgreSQLService) RestartPostgreSQL(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return err
	}

	if err := s.dockerSvc.RestartContainer(ctx, instance.ContainerID); err != nil {
		s.logger.Error("failed to restart container", zap.Error(err))
		return err
	}

	infra.Status = entities.StatusRunning
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     infra.UserID,
		Type:       "postgres_single",
		Action:     "restarted",
		Metadata: map[string]interface{}{
			"container_id": instance.ContainerID,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *postgreSQLService) DeletePostgreSQL(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return err
	}

	infra.Status = entities.StatusDeleting
	s.infraRepo.Update(infra)

	if err := s.dockerSvc.RemoveContainer(ctx, instance.ContainerID); err != nil {
		s.logger.Error("failed to remove container", zap.Error(err))
	}

	if instance.VolumeID != "" {
		if err := s.dockerSvc.RemoveVolume(ctx, instance.VolumeID); err != nil {
			s.logger.Error("failed to remove volume", zap.Error(err))
		}
	}

	if err := s.pgRepo.Delete(instance.ID); err != nil {
		s.logger.Error("failed to delete postgres instance", zap.Error(err))
	}

	infra.Status = entities.StatusDeleted
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     infra.UserID,
		Type:       "postgres_single",
		Action:     "deleted",
		Metadata: map[string]interface{}{
			"container_id": instance.ContainerID,
			"name":         infra.Name,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	s.logger.Info("postgresql deleted successfully", zap.String("instance_id", id))
	return nil
}

func (s *postgreSQLService) GetPostgreSQLInfo(ctx context.Context, id string) (*dto.PostgreSQLInfoResponse, error) {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return nil, err
	}

	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return nil, err
	}

	// Sync status from Docker container
	if instance.ContainerID != "" {
		if containerInfo, err := s.dockerSvc.InspectContainer(ctx, instance.ContainerID); err == nil {
			var newStatus entities.InfrastructureStatus
			if containerInfo.State.Running {
				newStatus = entities.StatusRunning
			} else if containerInfo.State.Dead {
				newStatus = entities.StatusFailed
			} else if containerInfo.State.ExitCode != 0 && containerInfo.State.ExitCode != -1 {
				newStatus = entities.StatusFailed
			} else {
				newStatus = entities.StatusStopped
			}
			if infra.Status != newStatus {
				infra.Status = newStatus
				s.infraRepo.Update(infra)
			}
		} else {
			// Container not found, mark as stopped
			if infra.Status != entities.StatusStopped && infra.Status != entities.StatusDeleted {
				infra.Status = entities.StatusStopped
				s.infraRepo.Update(infra)
			}
		}
	}

	return &dto.PostgreSQLInfoResponse{
		ID:           infra.ID,
		Name:         infra.Name,
		Status:       string(infra.Status),
		ContainerID:  instance.ContainerID,
		Version:      instance.Version,
		Port:         instance.Port,
		DatabaseName: instance.DatabaseName,
		Username:     instance.Username,
		CPULimit:     instance.CPULimit,
		MemoryLimit:  instance.MemoryLimit,
		StorageSize:  instance.StorageSize,
		CreatedAt:    infra.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    infra.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *postgreSQLService) GetPostgreSQLLogs(ctx context.Context, id string, tail string) (string, error) {
	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return "", err
	}

	tailNum := 100
	if tail != "" {
		if num, err := strconv.Atoi(tail); err == nil {
			tailNum = num
		}
	}

	logs, err := s.dockerSvc.GetContainerLogs(ctx, instance.ContainerID, tailNum)
	if err != nil {
		s.logger.Error("failed to get container logs", zap.Error(err))
		return "", err
	}

	return strings.Join(logs, "\n"), nil
}

func (s *postgreSQLService) GetPostgreSQLStats(ctx context.Context, id string) (*dto.PostgreSQLStatsResponse, error) {
	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return nil, err
	}

	statsReader, err := s.dockerSvc.GetContainerStats(ctx, instance.ContainerID)
	if err != nil {
		s.logger.Error("failed to get container stats", zap.Error(err))
		return nil, err
	}
	defer statsReader.Body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage    uint64 `json:"usage"`
			Limit    uint64 `json:"limit"`
			MaxUsage uint64 `json:"max_usage"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
		BlkioStats struct {
			IoServiceBytesRecursive []struct {
				Op    string `json:"op"`
				Value uint64 `json:"value"`
			} `json:"io_service_bytes_recursive"`
		} `json:"blkio_stats"`
	}

	if err := json.NewDecoder(statsReader.Body).Decode(&stats); err != nil {
		s.logger.Error("failed to decode stats", zap.Error(err))
		return nil, err
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)
	cpuPercent := 0.0
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * 100.0
	}

	memoryPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memoryPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	var networkRx, networkTx uint64
	for _, network := range stats.Networks {
		networkRx += network.RxBytes
		networkTx += network.TxBytes
	}

	var diskRead, diskWrite uint64
	for _, io := range stats.BlkioStats.IoServiceBytesRecursive {
		if io.Op == "Read" {
			diskRead += io.Value
		} else if io.Op == "Write" {
			diskWrite += io.Value
		}
	}

	return &dto.PostgreSQLStatsResponse{
		CPUPercent:    cpuPercent,
		MemoryUsed:    int64(stats.MemoryStats.Usage),
		MemoryLimit:   int64(stats.MemoryStats.Limit),
		MemoryPercent: memoryPercent,
		NetworkRx:     int64(networkRx),
		NetworkTx:     int64(networkTx),
		DiskRead:      int64(diskRead),
		DiskWrite:     int64(diskWrite),
	}, nil
}

func (s *postgreSQLService) BackupPostgreSQL(ctx context.Context, id string, req dto.BackupPostgreSQLRequest) (*dto.BackupPostgreSQLResponse, error) {
	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return nil, err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupFileName := fmt.Sprintf("backup-%s-%s.sql", instance.DatabaseName, timestamp)
	containerBackupPath := fmt.Sprintf("/tmp/%s", backupFileName)
	hostBackupPath := filepath.Join(req.BackupPath, backupFileName)

	if err := os.MkdirAll(req.BackupPath, 0755); err != nil {
		s.logger.Error("failed to create backup directory", zap.Error(err))
		return nil, err
	}

	s.logger.Info("starting pg_dump", zap.String("container", instance.ContainerID), zap.String("database", instance.DatabaseName))

	cmd := []string{
		"pg_dump",
		"-U", instance.Username,
		"-d", instance.DatabaseName,
		"-f", containerBackupPath,
	}

	output, err := s.dockerSvc.ExecCommand(ctx, instance.ContainerID, cmd)
	if err != nil {
		s.logger.Error("failed to execute pg_dump", zap.Error(err), zap.String("output", output))
		return nil, err
	}

	s.logger.Info("pg_dump completed, checking file in container")

	checkCmd := []string{"sh", "-c", fmt.Sprintf("ls -lh %s && cat %s", containerBackupPath, containerBackupPath)}
	checkOutput, err := s.dockerSvc.ExecCommand(ctx, instance.ContainerID, checkCmd)
	if err != nil {
		s.logger.Warn("failed to check backup file in container", zap.Error(err))
	}

	if err := os.WriteFile(hostBackupPath, []byte(checkOutput), 0644); err != nil {
		s.logger.Error("failed to write backup to host", zap.Error(err))
		return nil, err
	}

	fileInfo, err := os.Stat(hostBackupPath)
	fileSize := int64(0)
	if err == nil {
		fileSize = fileInfo.Size()
	}

	s.logger.Info("backup completed", zap.String("file", hostBackupPath), zap.Int64("size", fileSize))

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     instance.Infrastructure.UserID,
		Type:       "postgres_single",
		Action:     "backup_created",
		Metadata: map[string]interface{}{
			"backup_file": backupFileName,
			"size":        fileSize,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return &dto.BackupPostgreSQLResponse{
		BackupFile: hostBackupPath,
		Size:       fileSize,
	}, nil
}

func (s *postgreSQLService) RestorePostgreSQL(ctx context.Context, id string, req dto.RestorePostgreSQLRequest) error {
	instance, err := s.pgRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find postgres instance", zap.Error(err))
		return err
	}

	cmd := []string{
		"psql",
		"-U", instance.Username,
		"-d", instance.DatabaseName,
		"-f", req.BackupFile,
	}

	output, err := s.dockerSvc.ExecCommand(ctx, instance.ContainerID, cmd)
	if err != nil {
		s.logger.Error("failed to execute psql restore", zap.Error(err), zap.String("output", output))
		return err
	}

	s.logger.Info("restore completed", zap.String("file", req.BackupFile))

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     instance.Infrastructure.UserID,
		Type:       "postgres_single",
		Action:     "restored",
		Metadata: map[string]interface{}{
			"backup_file": req.BackupFile,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}
