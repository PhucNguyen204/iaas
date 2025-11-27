package repositories

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/PhucNguyen204/vcs-authentication-service/entities"
)

type UserRepoSuite struct {
	suite.Suite
	db   *gorm.DB
	repo IUserRepository
}

func (suite *UserRepoSuite) SetupTest() {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	assert.NoError(suite.T(), err)
	err = gormDB.AutoMigrate(&entities.User{})
	assert.NoError(suite.T(), err)
	suite.db = gormDB
	suite.repo = NewUserRepository(gormDB)
	suite.db.Create(&entities.User{
		ID:       "test-id",
		Username: "test-user",
		Hash:     "hashed-password",
		Email:    "test@example.com",
		Scopes: []*entities.UserScope{
			{Name: "user:read"},
			{Name: "user:write"},
		},
	})
}

func (suite *UserRepoSuite) TearDownTest() {
	sqlDB, err := suite.db.DB()
	assert.NoError(suite.T(), err)
	sqlDB.Close()
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, new(UserRepoSuite))
}

func (suite *UserRepoSuite) TestFindById() {
	found, err := suite.repo.FindById("test-id")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-user", found.Username)
}

func (suite *UserRepoSuite) TestFindByIdNotFound() {
	_, err := suite.repo.FindById("non-existent-id")
	assert.Error(suite.T(), err)
}

func (suite *UserRepoSuite) TestFindByName() {
	found, err := suite.repo.FindByName("test-user")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-user", found.Username)
}

func (suite *UserRepoSuite) TestFindByNameNotFound() {
	_, err := suite.repo.FindByName("unknown")
	assert.Error(suite.T(), err)
}

func (suite *UserRepoSuite) TestFindByEmail() {
	found, err := suite.repo.FindByEmail("test@example.com")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "test-user", found.Username)
}

func (suite *UserRepoSuite) TestFindByEmailNotFound() {
	_, err := suite.repo.FindByEmail("notfound@example.com")
	assert.Error(suite.T(), err)
}

func (suite *UserRepoSuite) TestUpdatePassword() {
	found, _ := suite.repo.FindById("test-id")
	assert.Equal(suite.T(), "hashed-password", found.Hash)

	err := suite.repo.UpdatePassword(found, "newhash")
	assert.NoError(suite.T(), err)

	updated, _ := suite.repo.FindById(found.ID)
	assert.Equal(suite.T(), "newhash", updated.Hash)
}

func (suite *UserRepoSuite) TestBeginTransactionError() {
	sqlDB, _ := suite.db.DB()
	sqlDB.Close()

	_, err := suite.repo.BeginTransaction(context.Background())
	assert.Error(suite.T(), err)
}

func (suite *UserRepoSuite) TestBeginAndWithTransaction_Rollback() {
	tx, err := suite.repo.BeginTransaction(suite.T().Context())
	assert.NoError(suite.T(), err)

	txRepo := suite.repo.WithTransaction(tx)

	_, err = txRepo.FindByName("test-user")
	assert.NoError(suite.T(), err)

	tx.Rollback()

	_, err = suite.repo.FindByName("ivan")
	assert.Error(suite.T(), err)
}
