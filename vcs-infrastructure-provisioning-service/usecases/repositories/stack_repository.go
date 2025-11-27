package repositories

import (
	"github.com/PhucNguyen204/vcs-infrastructure-provisioning-service/entities"
	"gorm.io/gorm"
)

type IStackRepository interface {
	// Stack CRUD
	Create(stack *entities.Stack) error
	FindByID(id string) (*entities.Stack, error)
	FindByUserID(userID string, limit, offset int) ([]entities.Stack, int64, error)
	Update(stack *entities.Stack) error
	Delete(id string) error

	// Stack Resources
	CreateResource(resource *entities.StackResource) error
	FindResourcesByStackID(stackID string) ([]entities.StackResource, error)
	FindResourceByInfrastructureID(infraID string) (*entities.StackResource, error)
	DeleteResource(id string) error
	DeleteResourcesByStackID(stackID string) error

	// Stack Templates
	CreateTemplate(template *entities.StackTemplate) error
	FindTemplateByID(id string) (*entities.StackTemplate, error)
	FindTemplatesByCategory(category string) ([]entities.StackTemplate, error)
	FindPublicTemplates() ([]entities.StackTemplate, error)

	// Stack Operations
	CreateOperation(operation *entities.StackOperation) error
	FindOperationByID(id string) (*entities.StackOperation, error)
	FindOperationsByStackID(stackID string) ([]entities.StackOperation, error)
	UpdateOperation(operation *entities.StackOperation) error
	DeleteOperationsByStackID(stackID string) error
}

type stackRepository struct {
	db *gorm.DB
}

func NewStackRepository(db *gorm.DB) IStackRepository {
	return &stackRepository{db: db}
}

// Stack CRUD
func (r *stackRepository) Create(stack *entities.Stack) error {
	return r.db.Create(stack).Error
}

func (r *stackRepository) FindByID(id string) (*entities.Stack, error) {
	var stack entities.Stack
	err := r.db.Preload("Resources").Preload("Resources.Infrastructure").First(&stack, "id = ?", id).Error
	return &stack, err
}

func (r *stackRepository) FindByUserID(userID string, limit, offset int) ([]entities.Stack, int64, error) {
	var stacks []entities.Stack
	var count int64

	query := r.db.Model(&entities.Stack{}).Where("user_id = ?", userID)
	query.Count(&count)

	err := query.Preload("Resources").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&stacks).Error

	return stacks, count, err
}

func (r *stackRepository) Update(stack *entities.Stack) error {
	return r.db.Save(stack).Error
}

func (r *stackRepository) Delete(id string) error {
	return r.db.Delete(&entities.Stack{}, "id = ?", id).Error
}

// Stack Resources
func (r *stackRepository) CreateResource(resource *entities.StackResource) error {
	return r.db.Create(resource).Error
}

func (r *stackRepository) FindResourcesByStackID(stackID string) ([]entities.StackResource, error) {
	var resources []entities.StackResource
	err := r.db.Preload("Infrastructure").
		Where("stack_id = ?", stackID).
		Order(`"order" ASC`).
		Find(&resources).Error
	return resources, err
}

func (r *stackRepository) FindResourceByInfrastructureID(infraID string) (*entities.StackResource, error) {
	var resource entities.StackResource
	err := r.db.First(&resource, "infrastructure_id = ?", infraID).Error
	return &resource, err
}

func (r *stackRepository) DeleteResource(id string) error {
	return r.db.Delete(&entities.StackResource{}, "id = ?", id).Error
}

func (r *stackRepository) DeleteResourcesByStackID(stackID string) error {
	return r.db.Where("stack_id = ?", stackID).Delete(&entities.StackResource{}).Error
}

// Stack Templates
func (r *stackRepository) CreateTemplate(template *entities.StackTemplate) error {
	return r.db.Create(template).Error
}

func (r *stackRepository) FindTemplateByID(id string) (*entities.StackTemplate, error) {
	var template entities.StackTemplate
	err := r.db.First(&template, "id = ?", id).Error
	return &template, err
}

func (r *stackRepository) FindTemplatesByCategory(category string) ([]entities.StackTemplate, error) {
	var templates []entities.StackTemplate
	err := r.db.Where("category = ? AND is_public = ?", category, true).Find(&templates).Error
	return templates, err
}

func (r *stackRepository) FindPublicTemplates() ([]entities.StackTemplate, error) {
	var templates []entities.StackTemplate
	err := r.db.Where("is_public = ?", true).Find(&templates).Error
	return templates, err
}

// Stack Operations
func (r *stackRepository) CreateOperation(operation *entities.StackOperation) error {
	return r.db.Create(operation).Error
}

func (r *stackRepository) FindOperationByID(id string) (*entities.StackOperation, error) {
	var operation entities.StackOperation
	err := r.db.First(&operation, "id = ?", id).Error
	return &operation, err
}

func (r *stackRepository) FindOperationsByStackID(stackID string) ([]entities.StackOperation, error) {
	var operations []entities.StackOperation
	err := r.db.Where("stack_id = ?", stackID).Order("started_at DESC").Find(&operations).Error
	return operations, err
}

func (r *stackRepository) UpdateOperation(operation *entities.StackOperation) error {
	return r.db.Save(operation).Error
}

func (r *stackRepository) DeleteOperationsByStackID(stackID string) error {
	return r.db.Where("stack_id = ?", stackID).Delete(&entities.StackOperation{}).Error
}
