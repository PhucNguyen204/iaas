package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/dto"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/usecases/repositories"
	"github.com/google/uuid"
)

type IStackService interface {
	CreateStack(ctx context.Context, userID string, req dto.CreateStackRequest) (*dto.StackInfo, error)
	GetStack(ctx context.Context, stackID string) (*dto.StackInfo, error)
	ListStacks(ctx context.Context, userID string, page, pageSize int) (*dto.StackListResponse, error)
	UpdateStack(ctx context.Context, stackID string, req dto.UpdateStackRequest) (*dto.StackInfo, error)
	DeleteStack(ctx context.Context, stackID string) error
	CloneStack(ctx context.Context, userID string, req dto.CloneStackRequest) (*dto.StackInfo, error)

	// Operations
	StartStack(ctx context.Context, stackID string) error
	StopStack(ctx context.Context, stackID string) error
	RestartStack(ctx context.Context, stackID string) error

	// Templates
	CreateTemplate(ctx context.Context, userID string, req dto.CreateStackTemplateRequest) (*dto.StackTemplateInfo, error)
	GetTemplate(ctx context.Context, templateID string) (*dto.StackTemplateInfo, error)
	ListPublicTemplates(ctx context.Context) ([]dto.StackTemplateInfo, error)
}

type stackService struct {
	stackRepo           repositories.IStackRepository
	infraRepo           repositories.IInfrastructureRepository
	nginxService        INginxService
	pgService           IPostgreSQLService
	clusterService      IPostgreSQLClusterService
	clusterRepo         repositories.IPostgreSQLClusterRepository
	pgDbService         IPostgresDatabaseService
	dockerService       IDockerServiceService
	nginxClusterService INginxClusterService
	nginxClusterRepo    repositories.INginxClusterRepository
	dindService         IDinDService
}

func NewStackService(
	stackRepo repositories.IStackRepository,
	infraRepo repositories.IInfrastructureRepository,
	nginxService INginxService,
	pgService IPostgreSQLService,
	clusterService IPostgreSQLClusterService,
	clusterRepo repositories.IPostgreSQLClusterRepository,
	pgDbService IPostgresDatabaseService,
	dockerService IDockerServiceService,
	nginxClusterService INginxClusterService,
	nginxClusterRepo repositories.INginxClusterRepository,
	dindService IDinDService,
) IStackService {
	return &stackService{
		stackRepo:           stackRepo,
		infraRepo:           infraRepo,
		nginxService:        nginxService,
		pgService:           pgService,
		clusterService:      clusterService,
		clusterRepo:         clusterRepo,
		pgDbService:         pgDbService,
		dockerService:       dockerService,
		nginxClusterService: nginxClusterService,
		nginxClusterRepo:    nginxClusterRepo,
		dindService:         dindService,
	}
}

func (s *stackService) CreateStack(ctx context.Context, userID string, req dto.CreateStackRequest) (*dto.StackInfo, error) {
	stackID := uuid.New().String()

	// Create stack record
	tagsJSON, _ := json.Marshal(req.Tags)
	stack := &entities.Stack{
		ID:          stackID,
		Name:        req.Name,
		Description: req.Description,
		Environment: req.Environment,
		ProjectID:   req.ProjectID,
		TenantID:    req.TenantID,
		UserID:      userID,
		Status:      entities.StackStatusCreating,
		Tags:        string(tagsJSON),
	}

	if err := s.stackRepo.Create(stack); err != nil {
		return nil, fmt.Errorf("failed to create stack: %w", err)
	}

	operation := &entities.StackOperation{
		ID:            uuid.New().String(),
		StackID:       stackID,
		OperationType: "CREATE",
		Status:        "IN_PROGRESS",
		UserID:        userID,
		Details:       "{}",
	}
	s.stackRepo.CreateOperation(operation)

	resourceMap := make(map[string]string) // name -> infrastructure_id

	for _, resInput := range req.Resources {
		infraID, err := s.createResource(ctx, userID, stackID, resInput, resourceMap)
		if err != nil {
			stack.Status = entities.StackStatusFailed
			s.stackRepo.Update(stack)

			operation.Status = "FAILED"
			operation.ErrorMessage = err.Error()
			now := time.Now()
			operation.CompletedAt = &now
			s.stackRepo.UpdateOperation(operation)

			return nil, fmt.Errorf("failed to create resource %s: %w", resInput.Name, err)
		}

		resourceMap[resInput.Name] = infraID

		// Create stack resource link
		dependsOnJSON, _ := json.Marshal(resInput.DependsOn)
		stackResource := &entities.StackResource{
			ID:               uuid.New().String(),
			StackID:          stackID,
			InfrastructureID: infraID,
			ResourceType:     resInput.Type,
			Role:             resInput.Role,
			DependsOn:        string(dependsOnJSON),
			Order:            resInput.Order,
		}

		if err := s.stackRepo.CreateResource(stackResource); err != nil {
			return nil, fmt.Errorf("failed to link resource: %w", err)
		}
	}

	// Update stack status
	stack.Status = entities.StackStatusRunning
	s.stackRepo.Update(stack)

	operation.Status = "COMPLETED"
	now := time.Now()
	operation.CompletedAt = &now
	s.stackRepo.UpdateOperation(operation)

	return s.GetStack(ctx, stackID)
}

func (s *stackService) createResource(ctx context.Context, userID, stackID string, resInput dto.CreateStackResourceInput, resourceMap map[string]string) (string, error) {
	specJSON, _ := json.Marshal(resInput.Spec)

	switch resInput.Type {
	case "NGINX_GATEWAY":
		var nginxReq dto.CreateNginxRequest
		if err := json.Unmarshal(specJSON, &nginxReq); err != nil {
			return "", err
		}
		nginxReq.Name = resInput.Name

		// Set defaults if not specified
		if nginxReq.Port == 0 {
			nginxReq.Port = 8080
		}
		if nginxReq.Config == "" {
			nginxReq.Config = "default"
		}

		resp, err := s.nginxService.CreateNginx(ctx, userID, nginxReq)
		if err != nil {
			return "", err
		}
		return resp.ID, nil

	case "POSTGRES_INSTANCE":
		var pgReq dto.CreatePostgreSQLRequest
		if err := json.Unmarshal(specJSON, &pgReq); err != nil {
			return "", err
		}
		pgReq.Name = resInput.Name

		// Set defaults if not specified
		if pgReq.Version == "" {
			pgReq.Version = "16-alpine"
		}
		if pgReq.Port == 0 {
			// Use random port in range 15432-25432 to avoid conflicts
			pgReq.Port = 15432 + int(uuid.New().ID()%10000)
		}
		if pgReq.DatabaseName == "" {
			pgReq.DatabaseName = "app"
		}
		if pgReq.Username == "" {
			pgReq.Username = "postgres"
		}
		if pgReq.Password == "" {
			pgReq.Password = "postgres123"
		}

		resp, err := s.pgService.CreatePostgreSQL(ctx, userID, pgReq)
		if err != nil {
			return "", err
		}
		return resp.ID, nil

	case "POSTGRES_DATABASE":
		var dbReq dto.CreateDatabaseRequest
		if err := json.Unmarshal(specJSON, &dbReq); err != nil {
			return "", err
		}
		dbReq.DBName = resInput.Name

		// Resolve instance_id from dependencies
		var instanceID string
		if len(resInput.DependsOn) > 0 {
			if id, ok := resourceMap[resInput.DependsOn[0]]; ok {
				instanceID = id
			}
		}

		resp, err := s.pgDbService.CreateDatabase(ctx, instanceID, dbReq)
		if err != nil {
			return "", err
		}
		return resp.ID, nil

	case "POSTGRES_CLUSTER":
		var clusterReq dto.CreateClusterRequest
		if err := json.Unmarshal(specJSON, &clusterReq); err != nil {
			return "", err
		}
		clusterReq.ClusterName = resInput.Name

		// Set defaults if not specified
		if clusterReq.NodeCount == 0 {
			clusterReq.NodeCount = 2
		}
		if clusterReq.PostgreSQLPassword == "" {
			clusterReq.PostgreSQLPassword = "postgres123"
		}
		if clusterReq.PostgreSQLVersion == "" {
			clusterReq.PostgreSQLVersion = "17"
		}
		if clusterReq.ReplicationMode == "" {
			clusterReq.ReplicationMode = "async"
		}
		if clusterReq.Namespace == "" {
			clusterReq.Namespace = "default"
		}
		if clusterReq.CPUPerNode == 0 {
			clusterReq.CPUPerNode = 1
		}
		if clusterReq.MemoryPerNode == 0 {
			clusterReq.MemoryPerNode = 512
		}

		resp, err := s.clusterService.CreateCluster(ctx, userID, clusterReq)
		if err != nil {
			return "", err
		}
		return resp.InfrastructureID, nil

	case "DOCKER_SERVICE":
		var dockerReq dto.CreateDockerServiceRequest
		if err := json.Unmarshal(specJSON, &dockerReq); err != nil {
			return "", err
		}
		dockerReq.Name = resInput.Name

		// Resolve dependencies for env vars (e.g., database connection strings)
		if len(resInput.DependsOn) > 0 {
			for _, depName := range resInput.DependsOn {
				if depID, ok := resourceMap[depName]; ok {
					// Get infrastructure to inject connection info
					infra, _ := s.infraRepo.FindByID(depID)
					if infra != nil && infra.Type == entities.TypePostgreSQLSingle {
						pgInstance, _ := s.pgService.GetPostgreSQLInfo(ctx, depID)
						if pgInstance != nil {
							dockerReq.EnvVars = append(dockerReq.EnvVars, dto.EnvVarInput{
								Key:      "DATABASE_HOST",
								Value:    pgInstance.Name,
								IsSecret: false,
							})
							dockerReq.EnvVars = append(dockerReq.EnvVars, dto.EnvVarInput{
								Key:      "DATABASE_PORT",
								Value:    fmt.Sprintf("%d", pgInstance.Port),
								IsSecret: false,
							})
						}
					}
				}
			}
		}

		resp, err := s.dockerService.CreateDockerService(ctx, userID, dockerReq)
		if err != nil {
			return "", err
		}
		return resp.InfrastructureID, nil

	case "NGINX_CLUSTER":
		var clusterReq dto.CreateNginxClusterRequest
		if err := json.Unmarshal(specJSON, &clusterReq); err != nil {
			return "", err
		}
		clusterReq.ClusterName = resInput.Name

		if clusterReq.NodeCount == 0 {
			clusterReq.NodeCount = 2
		}
		if clusterReq.HTTPPort == 0 {
			clusterReq.HTTPPort = 8080
		}
		if clusterReq.HealthCheckPath == "" {
			clusterReq.HealthCheckPath = "/health"
		}

		resp, err := s.nginxClusterService.CreateCluster(ctx, userID, clusterReq)
		if err != nil {
			return "", err
		}
		return resp.InfrastructureID, nil

	case "DIND_ENVIRONMENT":
		var dindReq dto.CreateDinDEnvironmentRequest
		if err := json.Unmarshal(specJSON, &dindReq); err != nil {
			return "", err
		}
		dindReq.Name = resInput.Name

		// Set defaults
		if dindReq.ResourcePlan == "" {
			dindReq.ResourcePlan = "medium"
		}

		resp, err := s.dindService.CreateEnvironment(ctx, userID, dindReq)
		if err != nil {
			return "", err
		}
		return resp.InfrastructureID, nil

	default:
		return "", fmt.Errorf("unsupported resource type: %s", resInput.Type)
	}
}

func (s *stackService) getPostgresClusterIDByInfra(infraID string) (string, error) {
	cluster, err := s.clusterRepo.FindByInfrastructureID(infraID)
	if err != nil {
		return "", err
	}
	return cluster.ID, nil
}

func (s *stackService) getNginxClusterIDByInfra(infraID string) (string, error) {
	cluster, err := s.nginxClusterRepo.FindByInfrastructureID(infraID)
	if err != nil {
		return "", err
	}
	return cluster.ID, nil
}

func (s *stackService) GetStack(ctx context.Context, stackID string) (*dto.StackInfo, error) {
	stack, err := s.stackRepo.FindByID(stackID)
	if err != nil {
		return nil, err
	}

	var tags []string
	json.Unmarshal([]byte(stack.Tags), &tags)

	resources, _ := s.stackRepo.FindResourcesByStackID(stackID)
	resourceInfos := []dto.StackResourceInfo{}

	for _, res := range resources {
		var dependsOn []string
		json.Unmarshal([]byte(res.DependsOn), &dependsOn)

		outputs := s.getResourceOutputs(ctx, res.ResourceType, res.InfrastructureID)

		resourceInfos = append(resourceInfos, dto.StackResourceInfo{
			ID:               res.ID,
			StackID:          res.StackID,
			InfrastructureID: res.InfrastructureID,
			ResourceType:     res.ResourceType,
			ResourceName:     res.Infrastructure.Name,
			Role:             res.Role,
			Status:           string(res.Infrastructure.Status),
			DependsOn:        dependsOn,
			Order:            res.Order,
			Outputs:          outputs,
			CreatedAt:        res.CreatedAt,
		})
	}

	return &dto.StackInfo{
		ID:          stack.ID,
		Name:        stack.Name,
		Description: stack.Description,
		Environment: stack.Environment,
		ProjectID:   stack.ProjectID,
		TenantID:    stack.TenantID,
		UserID:      stack.UserID,
		Status:      string(stack.Status),
		Tags:        tags,
		Resources:   resourceInfos,
		CreatedAt:   stack.CreatedAt,
		UpdatedAt:   stack.UpdatedAt,
	}, nil
}

func (s *stackService) getResourceOutputs(ctx context.Context, resourceType, infraID string) map[string]interface{} {
	outputs := make(map[string]interface{})

	switch resourceType {
	case "NGINX_GATEWAY":
		if nginx, err := s.nginxService.GetNginxInfo(ctx, infraID); err == nil {
			outputs["nginx_name"] = nginx.Name
			outputs["port"] = nginx.Port
			outputs["status"] = nginx.Status
		}
	case "POSTGRES_INSTANCE":
		if pg, err := s.pgService.GetPostgreSQLInfo(ctx, infraID); err == nil {
			outputs["connection_string"] = fmt.Sprintf("postgresql://%s:%s@%s:%d/%s",
				pg.Username, "****", pg.Name, pg.Port, pg.DatabaseName)
			outputs["host"] = pg.Name
			outputs["port"] = pg.Port
		}
	case "POSTGRES_CLUSTER":
		// Get cluster by infrastructure_id first, then get cluster info by cluster_id
		if clusterEntity, err := s.clusterRepo.FindByInfrastructureID(infraID); err == nil {
			if cluster, err := s.clusterService.GetClusterInfo(ctx, clusterEntity.ID); err == nil {
				outputs["cluster_id"] = clusterEntity.ID
				outputs["cluster_name"] = cluster.ClusterName
				outputs["node_count"] = len(cluster.Nodes)
				outputs["write_endpoint"] = fmt.Sprintf("%s:%d", cluster.WriteEndpoint.Host, cluster.WriteEndpoint.Port)
				outputs["replication_mode"] = cluster.ReplicationMode
				outputs["status"] = cluster.Status
				// Add node summary
				primaryCount := 0
				replicaCount := 0
				healthyCount := 0
				for _, node := range cluster.Nodes {
					if node.Role == "primary" {
						primaryCount++
					} else if node.Role == "replica" {
						replicaCount++
					}
					if node.IsHealthy {
						healthyCount++
					}
				}
				outputs["primary_nodes"] = primaryCount
				outputs["replica_nodes"] = replicaCount
				outputs["healthy_nodes"] = healthyCount
			}
		}
	case "DOCKER_SERVICE":
		if docker, err := s.dockerService.GetDockerService(ctx, infraID); err == nil {
			outputs["service_name"] = docker.Name
			outputs["status"] = docker.Status
		}
	case "NGINX_CLUSTER":
		if clusterEntity, err := s.nginxClusterRepo.FindByInfrastructureID(infraID); err == nil {
			if cluster, err := s.nginxClusterService.GetClusterInfo(ctx, clusterEntity.ID); err == nil {
				outputs["cluster_id"] = clusterEntity.ID
				outputs["cluster_name"] = cluster.ClusterName
				outputs["status"] = cluster.Status
				outputs["virtual_ip"] = cluster.VirtualIP
				outputs["node_count"] = cluster.NodeCount
				outputs["http_port"] = cluster.HTTPPort
				if cluster.HTTPSPort > 0 {
					outputs["https_port"] = cluster.HTTPSPort
				}
			}
		}
	case "DIND_ENVIRONMENT":
		if dindEnv, err := s.dindService.GetEnvironmentByInfraID(ctx, infraID); err == nil {
			outputs["environment_id"] = dindEnv.ID
			outputs["container_id"] = dindEnv.ContainerID
			outputs["docker_host"] = dindEnv.DockerHost
			outputs["ip_address"] = dindEnv.IPAddress
			outputs["resource_plan"] = dindEnv.ResourcePlan
			outputs["status"] = dindEnv.Status
		}
	}

	return outputs
}

func (s *stackService) ListStacks(ctx context.Context, userID string, page, pageSize int) (*dto.StackListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize
	stacks, total, err := s.stackRepo.FindByUserID(userID, pageSize, offset)
	if err != nil {
		return nil, err
	}

	summaries := []dto.StackSummary{}
	for _, stack := range stacks {
		var tags []string
		json.Unmarshal([]byte(stack.Tags), &tags)

		summaries = append(summaries, dto.StackSummary{
			ID:            stack.ID,
			Name:          stack.Name,
			Environment:   stack.Environment,
			Status:        string(stack.Status),
			ResourceCount: len(stack.Resources),
			Tags:          tags,
			CreatedAt:     stack.CreatedAt,
			UpdatedAt:     stack.UpdatedAt,
		})
	}

	return &dto.StackListResponse{
		Stacks:     summaries,
		TotalCount: int(total),
		Page:       page,
		PageSize:   pageSize,
	}, nil
}

func (s *stackService) UpdateStack(ctx context.Context, stackID string, req dto.UpdateStackRequest) (*dto.StackInfo, error) {
	stack, err := s.stackRepo.FindByID(stackID)
	if err != nil {
		return nil, err
	}

	// Update metadata
	if req.Name != "" {
		stack.Name = req.Name
	}
	if req.Description != "" {
		stack.Description = req.Description
	}
	if req.Tags != nil {
		tagsJSON, _ := json.Marshal(req.Tags)
		stack.Tags = string(tagsJSON)
	}

	stack.Status = entities.StackStatusUpdating
	s.stackRepo.Update(stack)

	// TODO: Handle add/remove/update resources

	stack.Status = entities.StackStatusRunning
	s.stackRepo.Update(stack)

	return s.GetStack(ctx, stackID)
}

func (s *stackService) DeleteStack(ctx context.Context, stackID string) error {
	stack, err := s.stackRepo.FindByID(stackID)
	if err != nil {
		return err
	}

	// Mark stack as deleting
	stack.Status = entities.StackStatusDeleting
	s.stackRepo.Update(stack)

	// Delete all infrastructure resources in reverse order
	resources, _ := s.stackRepo.FindResourcesByStackID(stackID)

	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		if err := s.deleteResource(ctx, res.ResourceType, res.InfrastructureID); err != nil {
			// Log error but continue deleting other resources
			fmt.Printf("Failed to delete resource %s (%s): %v\n", res.ResourceType, res.InfrastructureID, err)
		}
	}

	// Delete stack_resources records
	s.stackRepo.DeleteResourcesByStackID(stackID)

	// Mark stack as deleted (don't actually delete the record)
	stack.Status = entities.StackStatusDeleted
	s.stackRepo.Update(stack)

	return nil
}

func (s *stackService) deleteResource(ctx context.Context, resourceType, infraID string) error {
	switch resourceType {
	case "NGINX_GATEWAY":
		return s.nginxService.DeleteNginx(ctx, infraID)
	case "POSTGRES_INSTANCE":
		return s.pgService.DeletePostgreSQL(ctx, infraID)
	case "POSTGRES_CLUSTER":
		clusterID, err := s.getPostgresClusterIDByInfra(infraID)
		if err != nil {
			return err
		}
		return s.clusterService.DeleteCluster(ctx, clusterID)
	case "DOCKER_SERVICE":
		return s.dockerService.DeleteDockerService(ctx, infraID)
	case "NGINX_CLUSTER":
		clusterID, err := s.getNginxClusterIDByInfra(infraID)
		if err != nil {
			return err
		}
		return s.nginxClusterService.DeleteCluster(ctx, clusterID)
	case "DIND_ENVIRONMENT":
		// Get DinD environment by infrastructure ID
		dindEnv, err := s.dindService.GetEnvironmentByInfraID(ctx, infraID)
		if err != nil {
			return err
		}
		return s.dindService.DeleteEnvironment(ctx, dindEnv.ID)
	}
	return nil
}

func (s *stackService) CloneStack(ctx context.Context, userID string, req dto.CloneStackRequest) (*dto.StackInfo, error) {
	// Get source stack
	sourceStack, err := s.stackRepo.FindByID(req.SourceStackID)
	if err != nil {
		return nil, err
	}

	// Clone logic: create new stack with same resources but different names
	// TODO: Implement full clone logic

	return s.GetStack(ctx, sourceStack.ID)
}

func (s *stackService) StartStack(ctx context.Context, stackID string) error {
	resources, _ := s.stackRepo.FindResourcesByStackID(stackID)

	for _, res := range resources {
		switch res.ResourceType {
		case "POSTGRES_INSTANCE":
			s.pgService.StartPostgreSQL(ctx, res.InfrastructureID)
		case "POSTGRES_CLUSTER":
			if clusterID, err := s.getPostgresClusterIDByInfra(res.InfrastructureID); err == nil {
				s.clusterService.StartCluster(ctx, clusterID)
			}
		case "DOCKER_SERVICE":
			s.dockerService.StartDockerService(ctx, res.InfrastructureID)
		case "NGINX_GATEWAY":
			s.nginxService.StartNginx(ctx, res.InfrastructureID)
		case "NGINX_CLUSTER":
			if clusterID, err := s.getNginxClusterIDByInfra(res.InfrastructureID); err == nil {
				s.nginxClusterService.StartCluster(ctx, clusterID)
			}
		case "DIND_ENVIRONMENT":
			if dindEnv, err := s.dindService.GetEnvironmentByInfraID(ctx, res.InfrastructureID); err == nil {
				s.dindService.StartEnvironment(ctx, dindEnv.ID)
			}
		}
	}

	return nil
}

func (s *stackService) StopStack(ctx context.Context, stackID string) error {
	resources, _ := s.stackRepo.FindResourcesByStackID(stackID)

	// Stop in reverse order
	for i := len(resources) - 1; i >= 0; i-- {
		res := resources[i]
		switch res.ResourceType {
		case "NGINX_GATEWAY":
			s.nginxService.StopNginx(ctx, res.InfrastructureID)
		case "NGINX_CLUSTER":
			if clusterID, err := s.getNginxClusterIDByInfra(res.InfrastructureID); err == nil {
				s.nginxClusterService.StopCluster(ctx, clusterID)
			}
		case "DOCKER_SERVICE":
			s.dockerService.StopDockerService(ctx, res.InfrastructureID)
		case "POSTGRES_CLUSTER":
			if clusterID, err := s.getPostgresClusterIDByInfra(res.InfrastructureID); err == nil {
				s.clusterService.StopCluster(ctx, clusterID)
			}
		case "POSTGRES_INSTANCE":
			s.pgService.StopPostgreSQL(ctx, res.InfrastructureID)
		case "DIND_ENVIRONMENT":
			if dindEnv, err := s.dindService.GetEnvironmentByInfraID(ctx, res.InfrastructureID); err == nil {
				s.dindService.StopEnvironment(ctx, dindEnv.ID)
			}
		}
	}

	return nil
}

func (s *stackService) RestartStack(ctx context.Context, stackID string) error {
	if err := s.StopStack(ctx, stackID); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return s.StartStack(ctx, stackID)
}

func (s *stackService) CreateTemplate(ctx context.Context, userID string, req dto.CreateStackTemplateRequest) (*dto.StackTemplateInfo, error) {
	specJSON, _ := json.Marshal(req.Resources)

	template := &entities.StackTemplate{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		IsPublic:    req.IsPublic,
		UserID:      userID,
		Spec:        string(specJSON),
	}

	if err := s.stackRepo.CreateTemplate(template); err != nil {
		return nil, err
	}

	return s.GetTemplate(ctx, template.ID)
}

func (s *stackService) GetTemplate(ctx context.Context, templateID string) (*dto.StackTemplateInfo, error) {
	template, err := s.stackRepo.FindTemplateByID(templateID)
	if err != nil {
		return nil, err
	}

	var resources []dto.CreateStackResourceInput
	json.Unmarshal([]byte(template.Spec), &resources)

	return &dto.StackTemplateInfo{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Category:    template.Category,
		IsPublic:    template.IsPublic,
		UserID:      template.UserID,
		Resources:   resources,
		CreatedAt:   template.CreatedAt,
	}, nil
}

func (s *stackService) ListPublicTemplates(ctx context.Context) ([]dto.StackTemplateInfo, error) {
	templates, err := s.stackRepo.FindPublicTemplates()
	if err != nil {
		return nil, err
	}

	result := []dto.StackTemplateInfo{}
	for _, t := range templates {
		var resources []dto.CreateStackResourceInput
		json.Unmarshal([]byte(t.Spec), &resources)

		result = append(result, dto.StackTemplateInfo{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Category:    t.Category,
			IsPublic:    t.IsPublic,
			UserID:      t.UserID,
			Resources:   resources,
			CreatedAt:   t.CreatedAt,
		})
	}

	return result, nil
}
