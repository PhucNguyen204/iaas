package dto

type InfrastructureStatusUpdate struct {
	InfrastructureID string `json:"infrastructure_id"`
	ContainerID      string `json:"container_id"`
	Status           string `json:"status"`
	Action           string `json:"action"`
	Timestamp        string `json:"timestamp"`
}
