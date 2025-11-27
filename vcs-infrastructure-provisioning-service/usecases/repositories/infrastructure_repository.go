package repositories

import (
	"context"

	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"gorm.io/gorm"
)

type IInfrastructureRepository interface {
	Create(infra *entities.Infrastructure) error
	FindByID(id string) (*entities.Infrastructure, error)
	FindByUserID(userID string) ([]*entities.Infrastructure, error)
	FindByContainerID(ctx context.Context, containerID string) (*entities.Infrastructure, error)
	Update(infra *entities.Infrastructure) error
	Delete(id string) error
}

type infrastructureRepository struct {
	db *gorm.DB
}

func NewInfrastructureRepository(db *gorm.DB) IInfrastructureRepository {
	return &infrastructureRepository{db: db}
}

func (r *infrastructureRepository) Create(infra *entities.Infrastructure) error {
	return r.db.Create(infra).Error
}

func (r *infrastructureRepository) FindByID(id string) (*entities.Infrastructure, error) {
	var infra entities.Infrastructure
	if err := r.db.Where("id = ?", id).First(&infra).Error; err != nil {
		return nil, err
	}
	return &infra, nil
}

func (r *infrastructureRepository) FindByUserID(userID string) ([]*entities.Infrastructure, error) {
	var infras []*entities.Infrastructure
	if err := r.db.Where("user_id = ?", userID).Find(&infras).Error; err != nil {
		return nil, err
	}
	return infras, nil
}

func (r *infrastructureRepository) Update(infra *entities.Infrastructure) error {
	return r.db.Save(infra).Error
}

func (r *infrastructureRepository) Delete(id string) error {
	return r.db.Where("id = ?", id).Delete(&entities.Infrastructure{}).Error
}

func (r *infrastructureRepository) FindByContainerID(ctx context.Context, containerID string) (*entities.Infrastructure, error) {
	var infra entities.Infrastructure

	var pgInstance entities.PostgreSQLInstance
	if err := r.db.Where("container_id = ?", containerID).First(&pgInstance).Error; err == nil {
		if err := r.db.Where("id = ?", pgInstance.InfrastructureID).First(&infra).Error; err == nil {
			return &infra, nil
		}
	}

	var nginxInstance entities.NginxInstance
	if err := r.db.Where("container_id = ?", containerID).First(&nginxInstance).Error; err == nil {
		if err := r.db.Where("id = ?", nginxInstance.InfrastructureID).First(&infra).Error; err == nil {
			return &infra, nil
		}
	}

	var dockerService entities.DockerService
	if err := r.db.Where("container_id = ?", containerID).First(&dockerService).Error; err == nil {
		if err := r.db.Where("id = ?", dockerService.InfrastructureID).First(&infra).Error; err == nil {
			return &infra, nil
		}
	}

	var clusterNode entities.ClusterNode
	if err := r.db.Where("container_id = ?", containerID).First(&clusterNode).Error; err == nil {
		var cluster entities.PostgreSQLCluster
		if err := r.db.Where("id = ?", clusterNode.ClusterID).First(&cluster).Error; err == nil {
			if err := r.db.Where("id = ?", cluster.InfrastructureID).First(&infra).Error; err == nil {
				return &infra, nil
			}
		}
	}

	var etcdNode entities.EtcdNode
	if err := r.db.Where("container_id = ?", containerID).First(&etcdNode).Error; err == nil {
		var cluster entities.PostgreSQLCluster
		if err := r.db.Where("id = ?", etcdNode.ClusterID).First(&cluster).Error; err == nil {
			if err := r.db.Where("id = ?", cluster.InfrastructureID).First(&infra).Error; err == nil {
				return &infra, nil
			}
		}
	}

	return nil, gorm.ErrRecordNotFound
}
