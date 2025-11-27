package services

import (
	"context"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/kafka"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/docker/docker/api/types/events"
	"go.uber.org/zap"
)

type IDockerEventListenerService interface {
	Start(ctx context.Context) error
	Stop()
	SetWebSocketHandler(handler WebSocketBroadcaster)
}

type WebSocketBroadcaster interface {
	BroadcastUpdate(update dto.InfrastructureStatusUpdate)
}

type dockerEventListenerService struct {
	dockerService docker.IDockerService
	kafkaProducer kafka.IKafkaProducer
	infraRepo     repositories.IInfrastructureRepository
	logger        logger.ILogger
	eventChan     chan events.Message
	ctx           context.Context
	cancel        context.CancelFunc
	wsBroadcaster WebSocketBroadcaster
}

func NewDockerEventListenerService(
	dockerService docker.IDockerService,
	kafkaProducer kafka.IKafkaProducer,
	infraRepo repositories.IInfrastructureRepository,
	logger logger.ILogger,
) IDockerEventListenerService {
	ctx, cancel := context.WithCancel(context.Background())
	return &dockerEventListenerService{
		dockerService: dockerService,
		kafkaProducer: kafkaProducer,
		infraRepo:     infraRepo,
		logger:        logger,
		eventChan:     make(chan events.Message, 100),
		ctx:           ctx,
		cancel:        cancel,
	}
}

func (s *dockerEventListenerService) SetWebSocketHandler(handler WebSocketBroadcaster) {
	s.wsBroadcaster = handler
}

func (s *dockerEventListenerService) Start(ctx context.Context) error {
	// Start listening to Docker events
	if err := s.dockerService.ListenToEvents(ctx, s.eventChan); err != nil {
		s.logger.Error("failed to start docker event listener", zap.Error(err))
		return err
	}

	s.logger.Info("docker event listener started")

	// Process events in a goroutine
	go s.processEvents(ctx)

	return nil
}

func (s *dockerEventListenerService) Stop() {
	s.cancel()
	close(s.eventChan)
	s.logger.Info("docker event listener stopped")
}

func (s *dockerEventListenerService) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-s.eventChan:
			if !ok {
				return
			}
			s.handleEvent(ctx, event)
		}
	}
}

func (s *dockerEventListenerService) handleEvent(ctx context.Context, event events.Message) {
	containerID := event.ID
	action := string(event.Action)
	containerName := event.Actor.Attributes["name"]

	s.logger.Info("processing docker event",
		zap.String("action", action),
		zap.String("container_id", containerID),
		zap.String("container_name", containerName))

	// Map Docker actions to infrastructure status
	var status entities.InfrastructureStatus
	switch event.Action {
	case events.ActionStart:
		status = entities.StatusRunning
	case events.ActionStop, events.ActionDie:
		status = entities.StatusStopped
	case events.ActionDestroy, events.ActionRemove:
		status = entities.StatusDeleted
	case events.ActionCreate:
		status = entities.StatusCreating
	case events.ActionRestart:
		status = entities.StatusRunning
	default:
		// Skip unknown actions
		return
	}

	// Find infrastructure by container ID
	infra, err := s.infraRepo.FindByContainerID(ctx, containerID)
	if err != nil {
		// Container might not be managed by our system, skip
		s.logger.Debug("container not found in infrastructure",
			zap.String("container_id", containerID),
			zap.String("container_name", containerName))
		return
	}

	infra.Status = status
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status",
			zap.String("infrastructure_id", infra.ID),
			zap.String("container_id", containerID),
			zap.String("status", string(status)),
			zap.Error(err))
		return
	}

	s.logger.Info("infrastructure status updated",
		zap.String("infrastructure_id", infra.ID),
		zap.String("container_id", containerID),
		zap.String("status", string(status)))

	kafkaEvent := kafka.InfrastructureEvent{
		InstanceID: infra.ID,
		UserID:     infra.UserID,
		Type:       string(infra.Type),
		Action:     action,
		Timestamp:  time.Now(),
		Metadata: map[string]interface{}{
			"container_id":      containerID,
			"container_name":    containerName,
			"status":            string(status),
			"infrastructure_id": infra.ID,
		},
	}

	if err := s.kafkaProducer.PublishEvent(ctx, kafkaEvent); err != nil {
		s.logger.Error("failed to publish event to kafka",
			zap.String("infrastructure_id", infra.ID),
			zap.Error(err))
	}

	// Broadcast via WebSocket if handler is set
	if s.wsBroadcaster != nil {
		update := dto.InfrastructureStatusUpdate{
			InfrastructureID: infra.ID,
			ContainerID:      containerID,
			Status:           string(status),
			Action:           action,
			Timestamp:        time.Now().Format(time.RFC3339),
		}
		s.wsBroadcaster.BroadcastUpdate(update)
	}
}
