package dto

import "time"

type CreateStackRequest struct {
	Name         string                     `json:"name" binding:"required"`
	Description  string                     `json:"description"`
	Environment  string                     `json:"environment"` // dev, staging, prod
	ProjectID    string                     `json:"project_id"`
	TenantID     string                     `json:"tenant_id"`
	Tags         []string                   `json:"tags"`
	Resources    []CreateStackResourceInput `json:"resources" binding:"required"`
	FromTemplate string                     `json:"from_template"` // Optional template ID
}

type CreateStackResourceInput struct {
	Type      string                 `json:"resource_type" binding:"required"` // NGINX_GATEWAY, POSTGRES_INSTANCE, POSTGRES_DATABASE, POSTGRES_CLUSTER, DOCKER_SERVICE
	Role      string                 `json:"role"`                             // gateway, database, app, cache
	Name      string                 `json:"resource_name" binding:"required"`
	Spec      map[string]interface{} `json:"spec" binding:"required"` // Resource-specific config
	DependsOn []string               `json:"depends_on"`              // Array of resource names in this stack
	Order     int                    `json:"order"`                   // Creation order
}

type UpdateStackRequest struct {
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	Tags            []string                   `json:"tags"`
	AddResources    []CreateStackResourceInput `json:"add_resources"`    // Add new resources
	RemoveResources []string                   `json:"remove_resources"` // Infrastructure IDs to remove
	UpdateResources []UpdateStackResourceInput `json:"update_resources"` // Update existing resources
}

type UpdateStackResourceInput struct {
	InfrastructureID string                 `json:"infrastructure_id" binding:"required"`
	Role             string                 `json:"role"`
	Spec             map[string]interface{} `json:"spec"`
}

type CloneStackRequest struct {
	SourceStackID string   `json:"source_stack_id" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Environment   string   `json:"environment"`
	Tags          []string `json:"tags"`
}

type StackInfo struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Environment string              `json:"environment"`
	ProjectID   string              `json:"project_id"`
	TenantID    string              `json:"tenant_id"`
	UserID      string              `json:"user_id"`
	Status      string              `json:"status"`
	Tags        []string            `json:"tags"`
	Resources   []StackResourceInfo `json:"resources"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type StackResourceInfo struct {
	ID               string                 `json:"id"`
	StackID          string                 `json:"stack_id"`
	InfrastructureID string                 `json:"infrastructure_id"`
	ResourceType     string                 `json:"resource_type"`
	ResourceName     string                 `json:"resource_name"`
	Role             string                 `json:"role"`
	Status           string                 `json:"status"`
	DependsOn        []string               `json:"depends_on"`
	Order            int                    `json:"order"`
	Outputs          map[string]interface{} `json:"outputs"` // Connection strings, endpoints, etc.
	CreatedAt        time.Time              `json:"created_at"`
}

type StackListResponse struct {
	Stacks     []StackSummary `json:"stacks"`
	TotalCount int            `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
}

type StackSummary struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Environment   string    `json:"environment"`
	Status        string    `json:"status"`
	ResourceCount int       `json:"resource_count"`
	Tags          []string  `json:"tags"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// StackOperationRequest for start/stop/restart stack
type StackOperationRequest struct {
	Operation string `json:"operation" binding:"required"` // start, stop, restart
}

// CreateStackTemplateRequest creates a reusable template
type CreateStackTemplateRequest struct {
	Name        string                     `json:"name" binding:"required"`
	Description string                     `json:"description"`
	Category    string                     `json:"category"` // web-app, microservice, data-pipeline
	IsPublic    bool                       `json:"is_public"`
	Resources   []CreateStackResourceInput `json:"resources" binding:"required"`
}

// StackTemplateInfo represents a template
type StackTemplateInfo struct {
	ID          string                     `json:"id"`
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	Category    string                     `json:"category"`
	IsPublic    bool                       `json:"is_public"`
	UserID      string                     `json:"user_id"`
	Resources   []CreateStackResourceInput `json:"resources"`
	CreatedAt   time.Time                  `json:"created_at"`
}

// StackOperationInfo represents an operation on a stack
type StackOperationInfo struct {
	ID            string     `json:"id"`
	StackID       string     `json:"stack_id"`
	OperationType string     `json:"operation_type"`
	Status        string     `json:"status"`
	UserID        string     `json:"user_id"`
	StartedAt     time.Time  `json:"started_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	Details       string     `json:"details,omitempty"`
}
