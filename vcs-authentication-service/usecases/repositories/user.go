package repositories

import (
	"context"

	"github.com/PhucNguyen204/vcs-authentication-service/entities"

	"gorm.io/gorm"
)

type IUserRepository interface {
	FindById(userId string) (*entities.User, error)
	FindByName(username string) (*entities.User, error)
	FindByEmail(email string) (*entities.User, error)
	UpdatePassword(user *entities.User, hash string) error
	BeginTransaction(ctx context.Context) (*gorm.DB, error)
	WithTransaction(tx *gorm.DB) IUserRepository
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) IUserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindById(userId string) (*entities.User, error) {
	var user entities.User
	res := r.db.Preload("Scopes").First(&user, entities.User{ID: userId})
	if res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *userRepository) FindByName(username string) (*entities.User, error) {
	var user entities.User
	res := r.db.Preload("Scopes").First(&user, entities.User{Username: username})
	if res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(email string) (*entities.User, error) {
	var user entities.User
	res := r.db.Preload("Scopes").First(&user, entities.User{Email: email})
	if res.Error != nil {
		return nil, res.Error
	}
	return &user, nil
}

func (r *userRepository) UpdatePassword(user *entities.User, hash string) error {
	res := r.db.Model(user).Update("hash", hash)
	return res.Error
}

func (r *userRepository) BeginTransaction(ctx context.Context) (*gorm.DB, error) {
	tx := r.db.Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tx, nil
}

func (r *userRepository) WithTransaction(tx *gorm.DB) IUserRepository {
	return &userRepository{db: tx}
}
