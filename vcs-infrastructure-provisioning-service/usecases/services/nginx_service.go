package services

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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

type INginxService interface {
	CreateNginx(ctx context.Context, userID string, req dto.CreateNginxRequest) (*dto.NginxInfoResponse, error)
	StartNginx(ctx context.Context, id string) error
	StopNginx(ctx context.Context, id string) error
	RestartNginx(ctx context.Context, id string) error
	DeleteNginx(ctx context.Context, id string) error
	GetNginxInfo(ctx context.Context, id string) (*dto.NginxInfoResponse, error)
	UpdateNginxConfig(ctx context.Context, id string, req dto.UpdateNginxConfigRequest) error

	AddDomain(ctx context.Context, id string, req dto.AddDomainRequest) error
	DeleteDomain(ctx context.Context, id, domain string) error
	AddRoute(ctx context.Context, id string, req dto.AddRouteRequest) (*dto.RouteInfo, error)
	UpdateRoute(ctx context.Context, id, routeID string, req dto.UpdateRouteRequest) error
	DeleteRoute(ctx context.Context, id, routeID string) error
	UploadCertificate(ctx context.Context, id string, req dto.UploadCertificateRequest) error
	GetCertificate(ctx context.Context, id string) (*dto.CertificateInfo, error)
	UpdateUpstreams(ctx context.Context, id string, req dto.UpdateUpstreamsRequest) error
	GetUpstreams(ctx context.Context, id string) ([]dto.UpstreamInfo, error)
	SetSecurityPolicy(ctx context.Context, id string, req dto.SecurityPolicyRequest) error
	GetSecurityPolicy(ctx context.Context, id string) (*dto.SecurityPolicy, error)
	DeleteSecurityPolicy(ctx context.Context, id string) error
	GetLogs(ctx context.Context, id string, tail int) (*dto.NginxLogsResponse, error)
	GetMetrics(ctx context.Context, id string) (*dto.NginxMetricsResponse, error)
	GetStats(ctx context.Context, id string) (*dto.NginxStatsResponse, error)
}

type nginxService struct {
	infraRepo     repositories.IInfrastructureRepository
	nginxRepo     repositories.INginxRepository
	dockerSvc     docker.IDockerService
	kafkaProducer kafka.IKafkaProducer
	logger        logger.ILogger
}

func NewNginxService(
	infraRepo repositories.IInfrastructureRepository,
	nginxRepo repositories.INginxRepository,
	dockerSvc docker.IDockerService,
	kafkaProducer kafka.IKafkaProducer,
	logger logger.ILogger,
) INginxService {
	return &nginxService{
		infraRepo:     infraRepo,
		nginxRepo:     nginxRepo,
		dockerSvc:     dockerSvc,
		kafkaProducer: kafkaProducer,
		logger:        logger,
	}
}

func (s *nginxService) CreateNginx(ctx context.Context, userID string, req dto.CreateNginxRequest) (*dto.NginxInfoResponse, error) {
	infraID := uuid.New().String()
	instanceID := uuid.New().String()

	infra := &entities.Infrastructure{
		ID:     infraID,
		Name:   req.Name,
		Type:   entities.TypeNginx,
		Status: entities.StatusCreating,
		UserID: userID,
	}

	if err := s.infraRepo.Create(infra); err != nil {
		s.logger.Error("failed to create infrastructure record", zap.Error(err))
		return nil, err
	}

	instance := &entities.NginxInstance{
		ID:               instanceID,
		InfrastructureID: infraID,
		Port:             req.Port,
		SSLPort:          req.SSLPort,
		Config:           req.Config,
		CPULimit:         req.CPULimit,
		MemoryLimit:      req.MemoryLimit,
	}

	if err := s.nginxRepo.Create(instance); err != nil {
		s.logger.Error("failed to create nginx instance record", zap.Error(err))
		return nil, err
	}

	volumeName := fmt.Sprintf("iaas-nginx-%s", instanceID)
	if err := s.dockerSvc.CreateVolume(ctx, volumeName); err != nil {
		s.logger.Error("failed to create volume", zap.Error(err))
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		return nil, err
	}

	instance.VolumeID = volumeName

	containerName := fmt.Sprintf("iaas-nginx-%s", instanceID)
	ports := map[string]string{
		"80": fmt.Sprintf("%d", req.Port),
	}
	if req.SSLPort > 0 {
		ports["443"] = fmt.Sprintf("%d", req.SSLPort)
	}

	containerConfig := docker.ContainerConfig{
		Name:  containerName,
		Image: "nginx:latest",
		Ports: ports,
		Volumes: map[string]string{
			volumeName: "/etc/nginx/conf.d",
		},
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

	if err := s.nginxRepo.Update(instance); err != nil {
		s.logger.Error("failed to update nginx instance", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: infraID,
		UserID:     userID,
		Type:       "nginx",
		Action:     "created",
		Metadata: map[string]interface{}{
			"name": req.Name,
			"port": req.Port,
		},
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	s.logger.Info("nginx created successfully",
		zap.String("instance_id", instanceID),
		zap.String("container_id", containerID))

	return &dto.NginxInfoResponse{
		ID:          infraID,
		Name:        infra.Name,
		Status:      string(infra.Status),
		ContainerID: containerID,
		Port:        req.Port,
		SSLPort:     req.SSLPort,
		Config:      req.Config,
		CPULimit:    req.CPULimit,
		MemoryLimit: req.MemoryLimit,
		CreatedAt:   infra.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   infra.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (s *nginxService) StartNginx(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
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
		Type:       "nginx",
		Action:     "started",
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *nginxService) StopNginx(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
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
		Type:       "nginx",
		Action:     "stopped",
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *nginxService) RestartNginx(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
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
		Type:       "nginx",
		Action:     "restarted",
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	return nil
}

func (s *nginxService) DeleteNginx(ctx context.Context, id string) error {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return err
	}

	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
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

	if err := s.nginxRepo.Delete(instance.ID); err != nil {
		s.logger.Error("failed to delete nginx instance", zap.Error(err))
	}

	infra.Status = entities.StatusDeleted
	if err := s.infraRepo.Update(infra); err != nil {
		s.logger.Error("failed to update infrastructure status", zap.Error(err))
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     infra.UserID,
		Type:       "nginx",
		Action:     "deleted",
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	s.logger.Info("nginx deleted successfully", zap.String("instance_id", id))
	return nil
}

func (s *nginxService) GetNginxInfo(ctx context.Context, id string) (*dto.NginxInfoResponse, error) {
	infra, err := s.infraRepo.FindByID(id)
	if err != nil {
		s.logger.Error("failed to find infrastructure", zap.Error(err))
		return nil, err
	}

	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
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

	response := &dto.NginxInfoResponse{
		ID:          infra.ID,
		Name:        infra.Name,
		Status:      string(infra.Status),
		ContainerID: instance.ContainerID,
		Port:        instance.Port,
		SSLPort:     instance.SSLPort,
		Config:      instance.Config,
		CPULimit:    instance.CPULimit,
		MemoryLimit: instance.MemoryLimit,
		CreatedAt:   infra.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   infra.UpdatedAt.Format(time.RFC3339),
	}

	if domains, err := s.nginxRepo.ListDomains(instance.ID); err == nil {
		for _, d := range domains {
			response.Domains = append(response.Domains, d.Domain)
		}
	}

	if routes, err := s.nginxRepo.ListRoutes(instance.ID); err == nil {
		for _, r := range routes {
			response.Routes = append(response.Routes, dto.RouteInfo{
				ID:       r.ID,
				Path:     r.Path,
				Backend:  r.Backend,
				Priority: r.Priority,
			})
		}
	}

	if upstreams, err := s.nginxRepo.ListUpstreams(instance.ID); err == nil {
		for _, u := range upstreams {
			var backends []dto.BackendServer
			for _, b := range u.Backends {
				backends = append(backends, dto.BackendServer{
					Address: b.Address,
					Weight:  b.Weight,
				})
			}
			response.Upstreams = append(response.Upstreams, dto.UpstreamInfo{
				Name:     u.Name,
				Policy:   u.Policy,
				Backends: backends,
			})
		}
	}

	if cert, err := s.nginxRepo.GetCertificate(instance.ID); err == nil && cert != nil {
		response.Certificate = &dto.CertificateInfo{
			Domain:    cert.Domain,
			Status:    cert.Status,
			ExpiresAt: cert.ExpiresAt.Format(time.RFC3339),
			Issuer:    cert.Issuer,
		}
	}

	if security, err := s.nginxRepo.GetSecurity(instance.ID); err == nil && security != nil {
		policy := &dto.SecurityPolicy{}
		if security.RateLimitRPS > 0 {
			policy.RateLimit = &dto.RateLimitConfig{
				RequestsPerSecond: security.RateLimitRPS,
				Burst:             security.RateLimitBurst,
				Path:              security.RateLimitPath,
			}
		}
		if security.AllowIPs != "" || security.DenyIPs != "" {
			policy.IPFilter = &dto.IPFilterConfig{
				AllowIPs: strings.Split(security.AllowIPs, ","),
				DenyIPs:  strings.Split(security.DenyIPs, ","),
			}
		}
		if security.BasicAuthUsername != "" {
			policy.BasicAuth = &dto.BasicAuthConfig{
				Username: security.BasicAuthUsername,
				Realm:    security.BasicAuthRealm,
			}
		}
		response.Security = policy
	}

	return response, nil
}

func (s *nginxService) UpdateNginxConfig(ctx context.Context, id string, req dto.UpdateNginxConfigRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		s.logger.Error("failed to find nginx instance", zap.Error(err))
		return err
	}

	instance.Config = req.Config
	if err := s.nginxRepo.Update(instance); err != nil {
		s.logger.Error("failed to update nginx config", zap.Error(err))
		return err
	}

	if err := s.dockerSvc.RestartContainer(ctx, instance.ContainerID); err != nil {
		s.logger.Error("failed to restart container after config update", zap.Error(err))
		return err
	}

	event := kafka.InfrastructureEvent{
		InstanceID: id,
		UserID:     instance.Infrastructure.UserID,
		Type:       "nginx",
		Action:     "config_updated",
	}
	s.kafkaProducer.PublishEvent(ctx, event)

	s.logger.Info("nginx config updated successfully", zap.String("instance_id", id))
	return nil
}

func (s *nginxService) AddDomain(ctx context.Context, id string, req dto.AddDomainRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	if err := s.nginxRepo.CreateDomain(&entities.NginxDomain{
		ID: uuid.New().String(), NginxID: instance.ID, Domain: req.Domain,
	}); err != nil {
		return err
	}
	return nil
}

func (s *nginxService) DeleteDomain(ctx context.Context, id, domain string) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	return s.nginxRepo.DeleteDomain(instance.ID, domain)
}

func (s *nginxService) AddRoute(ctx context.Context, id string, req dto.AddRouteRequest) (*dto.RouteInfo, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	route := &entities.NginxRoute{
		ID: uuid.New().String(), NginxID: instance.ID, Path: req.Path, Backend: req.Backend, Priority: req.Priority,
	}
	if err := s.nginxRepo.CreateRoute(route); err != nil {
		return nil, err
	}
	return &dto.RouteInfo{ID: route.ID, Path: route.Path, Backend: route.Backend, Priority: route.Priority}, nil
}

func (s *nginxService) UpdateRoute(ctx context.Context, id, routeID string, req dto.UpdateRouteRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	route, err := s.nginxRepo.GetRoute(routeID)
	if err != nil {
		return err
	}
	if route.NginxID != instance.ID {
		return fmt.Errorf("route does not belong to this nginx instance")
	}
	if req.Backend != "" {
		route.Backend = req.Backend
	}
	route.Priority = req.Priority
	return s.nginxRepo.UpdateRoute(route)
}

func (s *nginxService) DeleteRoute(ctx context.Context, id, routeID string) error {
	return s.nginxRepo.DeleteRoute(routeID)
}

func (s *nginxService) UploadCertificate(ctx context.Context, id string, req dto.UploadCertificateRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	block, _ := pem.Decode([]byte(req.Certificate))
	if block == nil {
		return fmt.Errorf("invalid certificate PEM")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	return s.nginxRepo.CreateOrUpdateCertificate(&entities.NginxCertificate{
		ID: uuid.New().String(), NginxID: instance.ID, Domain: req.Domain,
		Certificate: req.Certificate, PrivateKey: req.PrivateKey,
		Status: "valid", ExpiresAt: cert.NotAfter, Issuer: cert.Issuer.String(),
	})
}

func (s *nginxService) GetCertificate(ctx context.Context, id string) (*dto.CertificateInfo, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	cert, err := s.nginxRepo.GetCertificate(instance.ID)
	if err != nil {
		return nil, err
	}
	return &dto.CertificateInfo{
		Domain: cert.Domain, Status: cert.Status,
		ExpiresAt: cert.ExpiresAt.Format(time.RFC3339), Issuer: cert.Issuer,
	}, nil
}

func (s *nginxService) UpdateUpstreams(ctx context.Context, id string, req dto.UpdateUpstreamsRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	upstream := &entities.NginxUpstream{
		ID: uuid.New().String(), NginxID: instance.ID, Name: "backend", Policy: req.Policy,
	}
	if err := s.nginxRepo.CreateOrUpdateUpstream(upstream); err != nil {
		return err
	}
	s.nginxRepo.DeleteUpstreamBackends(upstream.ID)
	for _, backend := range req.Backends {
		s.nginxRepo.CreateUpstreamBackend(&entities.NginxUpstreamBackend{
			ID: uuid.New().String(), UpstreamID: upstream.ID, Address: backend.Address, Weight: backend.Weight,
		})
	}
	return nil
}

func (s *nginxService) GetUpstreams(ctx context.Context, id string) ([]dto.UpstreamInfo, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	upstreams, err := s.nginxRepo.ListUpstreams(instance.ID)
	if err != nil {
		return nil, err
	}
	result := make([]dto.UpstreamInfo, 0, len(upstreams))
	for _, u := range upstreams {
		backends := make([]dto.BackendServer, 0, len(u.Backends))
		for _, b := range u.Backends {
			backends = append(backends, dto.BackendServer{Address: b.Address, Weight: b.Weight})
		}
		result = append(result, dto.UpstreamInfo{Name: u.Name, Backends: backends, Policy: u.Policy})
	}
	return result, nil
}

func (s *nginxService) SetSecurityPolicy(ctx context.Context, id string, req dto.SecurityPolicyRequest) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	security := &entities.NginxSecurity{ID: uuid.New().String(), NginxID: instance.ID}
	if req.RateLimit != nil {
		security.RateLimitRPS = req.RateLimit.RequestsPerSecond
		security.RateLimitBurst = req.RateLimit.Burst
		security.RateLimitPath = req.RateLimit.Path
	}
	if req.IPFilter != nil {
		security.AllowIPs = strings.Join(req.IPFilter.AllowIPs, ",")
		security.DenyIPs = strings.Join(req.IPFilter.DenyIPs, ",")
	}
	if req.BasicAuth != nil {
		security.BasicAuthUsername = req.BasicAuth.Username
		security.BasicAuthPassword = req.BasicAuth.Password
		security.BasicAuthRealm = req.BasicAuth.Realm
	}
	return s.nginxRepo.CreateOrUpdateSecurity(security)
}

func (s *nginxService) GetSecurityPolicy(ctx context.Context, id string) (*dto.SecurityPolicy, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	security, err := s.nginxRepo.GetSecurity(instance.ID)
	if err != nil {
		return nil, err
	}
	policy := &dto.SecurityPolicy{}
	if security.RateLimitRPS > 0 {
		policy.RateLimit = &dto.RateLimitConfig{
			RequestsPerSecond: security.RateLimitRPS, Burst: security.RateLimitBurst, Path: security.RateLimitPath,
		}
	}
	if security.AllowIPs != "" || security.DenyIPs != "" {
		policy.IPFilter = &dto.IPFilterConfig{
			AllowIPs: strings.Split(security.AllowIPs, ","), DenyIPs: strings.Split(security.DenyIPs, ","),
		}
	}
	if security.BasicAuthUsername != "" {
		policy.BasicAuth = &dto.BasicAuthConfig{
			Username: security.BasicAuthUsername, Password: security.BasicAuthPassword, Realm: security.BasicAuthRealm,
		}
	}
	return policy, nil
}

func (s *nginxService) DeleteSecurityPolicy(ctx context.Context, id string) error {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return err
	}
	return s.nginxRepo.DeleteSecurity(instance.ID)
}

func (s *nginxService) GetLogs(ctx context.Context, id string, tail int) (*dto.NginxLogsResponse, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	logLines, err := s.dockerSvc.GetContainerLogs(ctx, instance.ContainerID, tail)
	if err != nil {
		return nil, err
	}
	if len(logLines) > tail {
		logLines = logLines[len(logLines)-tail:]
	}
	return &dto.NginxLogsResponse{InstanceID: id, Logs: logLines, Tail: tail}, nil
}

func (s *nginxService) GetMetrics(ctx context.Context, id string) (*dto.NginxMetricsResponse, error) {
	return &dto.NginxMetricsResponse{InstanceID: id}, nil
}

func (s *nginxService) GetStats(ctx context.Context, id string) (*dto.NginxStatsResponse, error) {
	instance, err := s.nginxRepo.FindByInfrastructureID(id)
	if err != nil {
		return nil, err
	}
	upstreams, _ := s.nginxRepo.ListUpstreams(instance.ID)
	upstreamHealth := make([]dto.UpstreamHealth, 0)
	for _, u := range upstreams {
		for _, b := range u.Backends {
			upstreamHealth = append(upstreamHealth, dto.UpstreamHealth{
				Name: u.Name, Address: b.Address, Healthy: true,
			})
		}
	}
	return &dto.NginxStatsResponse{InstanceID: id, UpstreamHealth: upstreamHealth}, nil
}
