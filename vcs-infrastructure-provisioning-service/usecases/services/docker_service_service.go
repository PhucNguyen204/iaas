package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/infrastructures/docker"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/google/uuid"
)

type IDockerServiceService interface {
	CreateDockerService(ctx context.Context, userID string, req dto.CreateDockerServiceRequest) (*dto.DockerServiceInfo, error)
	GetDockerService(ctx context.Context, serviceID string) (*dto.DockerServiceInfo, error)
	StartDockerService(ctx context.Context, serviceID string) error
	StopDockerService(ctx context.Context, serviceID string) error
	RestartDockerService(ctx context.Context, serviceID string) error
	DeleteDockerService(ctx context.Context, serviceID string) error
	UpdateEnvVars(ctx context.Context, serviceID string, req dto.UpdateDockerEnvRequest) error
	GetServiceLogs(ctx context.Context, serviceID string, tail int) ([]string, error)
}

type dockerServiceService struct {
	dockerRepo repositories.IDockerServiceRepository
	infraRepo  repositories.IInfrastructureRepository
	dockerSvc  docker.IDockerService
}

func NewDockerServiceService(
	dockerRepo repositories.IDockerServiceRepository,
	infraRepo repositories.IInfrastructureRepository,
	dockerSvc docker.IDockerService,
) IDockerServiceService {
	return &dockerServiceService{
		dockerRepo: dockerRepo,
		infraRepo:  infraRepo,
		dockerSvc:  dockerSvc,
	}
}

func (s *dockerServiceService) CreateDockerService(ctx context.Context, userID string, req dto.CreateDockerServiceRequest) (*dto.DockerServiceInfo, error) {
	infraID := uuid.New().String()
	serviceID := uuid.New().String()

	infra := &entities.Infrastructure{
		ID:     infraID,
		Name:   req.Name,
		Type:   entities.TypeDockerService,
		Status: entities.StatusCreating,
		UserID: userID,
	}
	if err := s.infraRepo.Create(infra); err != nil {
		return nil, err
	}

	cpuLimit, memoryLimit := s.getPlanResources(req.Plan)

	service := &entities.DockerService{
		ID:               serviceID,
		InfrastructureID: infraID,
		Name:             req.Name,
		Image:            req.Image,
		ImageTag:         req.ImageTag,
		ServiceType:      req.ServiceType,
		Command:          req.Command,
		Args:             req.Args,
		RestartPolicy:    req.RestartPolicy,
		CPULimit:         cpuLimit,
		MemoryLimit:      memoryLimit,
		Status:           "creating",
	}

	if service.ServiceType == "" {
		service.ServiceType = "web"
	}
	if service.RestartPolicy == "" {
		service.RestartPolicy = "unless-stopped"
	}

	if err := s.dockerRepo.Create(service); err != nil {
		return nil, err
	}

	for _, env := range req.EnvVars {
		envVar := &entities.DockerEnvVar{
			ID:        uuid.New().String(),
			ServiceID: serviceID,
			Key:       env.Key,
			Value:     env.Value,
			IsSecret:  env.IsSecret,
		}
		s.dockerRepo.CreateEnvVar(envVar)
	}

	for _, port := range req.Ports {
		dockerPort := &entities.DockerPort{
			ID:            uuid.New().String(),
			ServiceID:     serviceID,
			ContainerPort: port.ContainerPort,
			HostPort:      port.HostPort,
			Protocol:      port.Protocol,
		}
		if dockerPort.Protocol == "" {
			dockerPort.Protocol = "tcp"
		}
		s.dockerRepo.CreatePort(dockerPort)
	}

	if len(req.Networks) > 0 {
		for _, network := range req.Networks {
			dockerNetwork := &entities.DockerNetwork{
				ID:        uuid.New().String(),
				ServiceID: serviceID,
				NetworkID: network.NetworkID,
				Alias:     network.Alias,
			}
			s.dockerRepo.CreateNetwork(dockerNetwork)
		}
	}

	if req.HealthCheck != nil {
		healthCheck := &entities.DockerHealthCheck{
			ID:                 uuid.New().String(),
			ServiceID:          serviceID,
			Type:               req.HealthCheck.Type,
			HTTPPath:           req.HealthCheck.HTTPPath,
			Port:               req.HealthCheck.Port,
			Command:            req.HealthCheck.Command,
			Interval:           req.HealthCheck.Interval,
			Timeout:            req.HealthCheck.Timeout,
			HealthyThreshold:   req.HealthCheck.HealthyThreshold,
			UnhealthyThreshold: req.HealthCheck.UnhealthyThreshold,
			Status:             "unknown",
		}
		if healthCheck.Interval == 0 {
			healthCheck.Interval = 30
		}
		if healthCheck.Timeout == 0 {
			healthCheck.Timeout = 10
		}
		s.dockerRepo.CreateHealthCheck(healthCheck)
	}

	envVars := []string{}
	for _, env := range req.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", env.Key, env.Value))
	}

	ports := map[string]string{}
	for _, port := range req.Ports {
		containerPortStr := fmt.Sprintf("%d", port.ContainerPort)
		if port.HostPort > 0 {
			ports[containerPortStr] = fmt.Sprintf("%d", port.HostPort)
		}
	}

	networks := []string{"iaas_iaas-network"}
	if len(req.Networks) > 0 {
		networks = []string{}
		for _, network := range req.Networks {
			networks = append(networks, network.NetworkID)
		}
	}

	containerName := fmt.Sprintf("iaas-docker-%s", serviceID)
	containerConfig := docker.ContainerConfig{
		Name:    containerName,
		Image:   fmt.Sprintf("%s:%s", req.Image, req.ImageTag),
		Env:     envVars,
		Ports:   ports,
		Network: networks[0],
		Resources: docker.ResourceConfig{
			CPULimit:    cpuLimit,
			MemoryLimit: memoryLimit,
		},
	}

	if req.Command != "" {
		containerConfig.Cmd = strings.Split(req.Command, " ")
	}

	containerID, err := s.dockerSvc.CreateContainer(ctx, containerConfig)
	if err != nil {
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		service.Status = "failed"
		s.dockerRepo.Update(service)
		return nil, err
	}

	service.ContainerID = containerID
	service.ContainerName = containerName

	if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
		infra.Status = entities.StatusFailed
		s.infraRepo.Update(infra)
		service.Status = "failed"
		s.dockerRepo.Update(service)
		return nil, err
	}

	containerIP, err := s.getContainerIP(ctx, containerID)
	if err == nil {
		service.IPAddress = containerIP
		if len(req.Ports) > 0 {
			service.InternalEndpoint = fmt.Sprintf("%s:%d", containerIP, req.Ports[0].ContainerPort)
		}
	}

	service.Status = "running"
	s.dockerRepo.Update(service)

	infra.Status = entities.StatusRunning
	s.infraRepo.Update(infra)

	return s.GetDockerService(ctx, serviceID)
}

func (s *dockerServiceService) GetDockerService(ctx context.Context, serviceID string) (*dto.DockerServiceInfo, error) {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return nil, err
	}

	// Sync status from Docker container if infrastructure exists
	if service.InfrastructureID != "" && service.ContainerID != "" {
		infra, err := s.infraRepo.FindByID(service.InfrastructureID)
		if err == nil {
			if containerInfo, err := s.dockerSvc.InspectContainer(ctx, service.ContainerID); err == nil {
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
					// Also update service status
					service.Status = string(newStatus)
					s.dockerRepo.Update(service)
				}
			} else {
				// Container not found, mark as stopped
				if infra.Status != entities.StatusStopped && infra.Status != entities.StatusDeleted {
					infra.Status = entities.StatusStopped
					s.infraRepo.Update(infra)
					service.Status = "stopped"
					s.dockerRepo.Update(service)
				}
			}
		}
	}

	envVars := []dto.EnvVarInfo{}
	for _, env := range service.EnvVars {
		envInfo := dto.EnvVarInfo{
			Key:      env.Key,
			IsSecret: env.IsSecret,
		}
		if !env.IsSecret {
			envInfo.Value = env.Value
		}
		envVars = append(envVars, envInfo)
	}

	ports := []dto.PortInfo{}
	for _, port := range service.Ports {
		ports = append(ports, dto.PortInfo{
			ContainerPort: port.ContainerPort,
			HostPort:      port.HostPort,
			Protocol:      port.Protocol,
		})
	}

	networks := []dto.NetworkInfo{}
	for _, network := range service.Networks {
		networks = append(networks, dto.NetworkInfo{
			NetworkID: network.NetworkID,
			Alias:     network.Alias,
		})
	}

	var healthCheck *dto.HealthCheckInfo
	if service.HealthCheck != nil {
		healthCheck = &dto.HealthCheckInfo{
			Type:               service.HealthCheck.Type,
			HTTPPath:           service.HealthCheck.HTTPPath,
			Port:               service.HealthCheck.Port,
			Command:            service.HealthCheck.Command,
			Interval:           service.HealthCheck.Interval,
			Timeout:            service.HealthCheck.Timeout,
			HealthyThreshold:   service.HealthCheck.HealthyThreshold,
			UnhealthyThreshold: service.HealthCheck.UnhealthyThreshold,
			Status:             service.HealthCheck.Status,
		}
		if !service.HealthCheck.LastCheck.IsZero() {
			healthCheck.LastCheck = service.HealthCheck.LastCheck.Format("2006-01-02T15:04:05Z")
		}
	}

	return &dto.DockerServiceInfo{
		ID:               service.ID,
		InfrastructureID: service.InfrastructureID,
		Name:             service.Name,
		Image:            service.Image,
		ImageTag:         service.ImageTag,
		ServiceType:      service.ServiceType,
		ContainerID:      service.ContainerID,
		Status:           service.Status,
		IPAddress:        service.IPAddress,
		InternalEndpoint: service.InternalEndpoint,
		EnvVars:          envVars,
		Ports:            ports,
		Networks:         networks,
		HealthCheck:      healthCheck,
		RestartPolicy:    service.RestartPolicy,
		CPULimit:         service.CPULimit,
		MemoryLimit:      service.MemoryLimit,
		CreatedAt:        service.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:        service.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *dockerServiceService) StartDockerService(ctx context.Context, serviceID string) error {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return err
	}

	if err := s.dockerSvc.StartContainer(ctx, service.ContainerID); err != nil {
		return err
	}

	service.Status = "running"
	return s.dockerRepo.Update(service)
}

func (s *dockerServiceService) StopDockerService(ctx context.Context, serviceID string) error {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return err
	}

	if err := s.dockerSvc.StopContainer(ctx, service.ContainerID); err != nil {
		return err
	}

	service.Status = "stopped"
	return s.dockerRepo.Update(service)
}

func (s *dockerServiceService) RestartDockerService(ctx context.Context, serviceID string) error {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return err
	}

	if err := s.dockerSvc.RestartContainer(ctx, service.ContainerID); err != nil {
		return err
	}

	service.Status = "running"
	return s.dockerRepo.Update(service)
}

func (s *dockerServiceService) DeleteDockerService(ctx context.Context, serviceID string) error {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return err
	}

	if service.ContainerID != "" {
		s.dockerSvc.StopContainer(ctx, service.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, service.ContainerID)
	}

	if err := s.dockerRepo.Delete(serviceID); err != nil {
		return err
	}

	infra, err := s.infraRepo.FindByID(service.InfrastructureID)
	if err == nil {
		infra.Status = entities.StatusDeleted
		s.infraRepo.Update(infra)
	}

	return nil
}

func (s *dockerServiceService) UpdateEnvVars(ctx context.Context, serviceID string, req dto.UpdateDockerEnvRequest) error {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return err
	}

	envVars := []entities.DockerEnvVar{}
	for _, env := range req.EnvVars {
		envVars = append(envVars, entities.DockerEnvVar{
			ID:        uuid.New().String(),
			ServiceID: serviceID,
			Key:       env.Key,
			Value:     env.Value,
			IsSecret:  env.IsSecret,
		})
	}

	if err := s.dockerRepo.UpdateEnvVars(serviceID, envVars); err != nil {
		return err
	}

	if service.ContainerID != "" {
		s.dockerSvc.StopContainer(ctx, service.ContainerID)
		s.dockerSvc.RemoveContainer(ctx, service.ContainerID)

		envStrings := []string{}
		for _, env := range req.EnvVars {
			envStrings = append(envStrings, fmt.Sprintf("%s=%s", env.Key, env.Value))
		}

		ports := map[string]string{}
		for _, port := range service.Ports {
			containerPortStr := fmt.Sprintf("%d", port.ContainerPort)
			if port.HostPort > 0 {
				ports[containerPortStr] = fmt.Sprintf("%d", port.HostPort)
			}
		}

		containerConfig := docker.ContainerConfig{
			Name:    service.ContainerName,
			Image:   fmt.Sprintf("%s:%s", service.Image, service.ImageTag),
			Env:     envStrings,
			Ports:   ports,
			Network: "iaas_iaas-network",
			Resources: docker.ResourceConfig{
				CPULimit:    service.CPULimit,
				MemoryLimit: service.MemoryLimit,
			},
		}

		containerID, err := s.dockerSvc.CreateContainer(ctx, containerConfig)
		if err != nil {
			return err
		}

		service.ContainerID = containerID
		s.dockerRepo.Update(service)

		if err := s.dockerSvc.StartContainer(ctx, containerID); err != nil {
			return err
		}
	}

	return nil
}

func (s *dockerServiceService) GetServiceLogs(ctx context.Context, serviceID string, tail int) ([]string, error) {
	service, err := s.dockerRepo.FindByID(serviceID)
	if err != nil {
		return nil, err
	}

	if service.ContainerID == "" {
		return []string{}, nil
	}

	logs, err := s.dockerSvc.GetContainerLogs(ctx, service.ContainerID, tail)
	if err != nil {
		return nil, err
	}

	return logs, nil
}

func (s *dockerServiceService) getPlanResources(plan string) (int64, int64) {
	switch plan {
	case "small":
		return 500000000, 536870912
	case "medium":
		return 1000000000, 1073741824
	case "large":
		return 2000000000, 2147483648
	default:
		return 1000000000, 1073741824
	}
}

func (s *dockerServiceService) getContainerIP(ctx context.Context, containerID string) (string, error) {
	info, err := s.dockerSvc.InspectContainer(ctx, containerID)
	if err != nil {
		return "", err
	}
	if info.NetworkSettings.IPAddress != "" {
		return info.NetworkSettings.IPAddress, nil
	}
	for _, network := range info.NetworkSettings.Networks {
		if network.IPAddress != "" {
			return network.IPAddress, nil
		}
	}
	return "", fmt.Errorf("no IP address found")
}
