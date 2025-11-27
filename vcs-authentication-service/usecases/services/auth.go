package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/mail"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/PhucNguyen204/vcs-authentication-service/entities"
	"github.com/PhucNguyen204/vcs-authentication-service/interfaces"
	"github.com/PhucNguyen204/vcs-authentication-service/pkg/env"
	"github.com/PhucNguyen204/vcs-authentication-service/pkg/logger"
	"github.com/PhucNguyen204/vcs-authentication-service/usecases/repositories"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type IAuthService interface {
	Login(ctx context.Context, username, password string) (string, string, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (string, error)
	UpdatePassword(ctx context.Context, userId, currentPassword, newPassword string) error
}

type authService struct {
	userRepo    repositories.IUserRepository
	redisClient interfaces.IRedisClient
	logger      logger.ILogger
	jwtSecret   []byte
}

func NewAuthService(userRepo repositories.IUserRepository, redisClient interfaces.IRedisClient, logger logger.ILogger, env env.AuthEnv) IAuthService {
	return &authService{
		userRepo:    userRepo,
		redisClient: redisClient,
		logger:      logger,
		jwtSecret:   []byte(env.JWTSecret),
	}
}

func (s *authService) Login(ctx context.Context, username, password string) (string, string, error) {
	var user *entities.User
	mail, err := mail.ParseAddress(username)
	if err != nil {
		user, err = s.userRepo.FindByName(username)
		if err != nil {
			s.logger.Error("failed to find user by username", zap.Error(err))
			return "", "", err
		}
	} else {
		user, err = s.userRepo.FindByEmail(mail.Address)
		if err != nil {
			s.logger.Error("failed to find user by email", zap.Error(err))
			return "", "", err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Hash), []byte(password)); err != nil {
		s.logger.Error("failed to validate password", zap.Error(err))
		return "", "", err
	}

	scopes := make([]string, 0, len(user.Scopes))
	for _, scope := range user.Scopes {
		scopes = append(scopes, scope.Name)
	}

	accessToken, err := s.generateAccessToken(user.ID, scopes)
	if err != nil {
		s.logger.Error("failed to generate access token", zap.Error(err))
		return "", "", err
	}
	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Error(err))
		return "", "", err
	}

	if err := s.redisClient.Set(ctx, "refresh:"+refreshToken, user.ID, time.Hour*24*7); err != nil {
		s.logger.Error("failed to set refresh token in redis", zap.Error(err))
		return "", "", err
	}

	s.logger.Info("user logged in successfully")
	return accessToken, refreshToken, nil
}

func (s *authService) UpdatePassword(ctx context.Context, userId, currentPassword, newPassword string) error {
	user, err := s.userRepo.FindById(userId)
	if err != nil {
		s.logger.Error("failed to find user by id", zap.Error(err))
		return err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Hash), []byte(currentPassword)); err != nil {
		s.logger.Error("current password does not match", zap.Error(err))
		return err
	}

	newHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("failed to hash password", zap.Error(err))
		return err
	}

	if err := s.userRepo.UpdatePassword(user, string(newHash)); err != nil {
		s.logger.Error("failed to update user's password", zap.Error(err))
		return err
	}

	if err := s.redisClient.Del(ctx, "refresh:"+user.ID); err != nil {
		s.logger.Error("failed to delete refresh token in redis", zap.Error(err))
		return err
	}

	s.logger.Info("user's password updated successfully")
	return nil
}

func (s *authService) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	userId, err := s.redisClient.Get(ctx, "refresh:"+refreshToken)
	if err != nil {
		s.logger.Error("failed to get refresh token from redis", zap.Error(err))
		return "", err
	}

	user, err := s.userRepo.FindById(userId)
	if err != nil {
		s.logger.Error("failed to find user by id", zap.Error(err))
		return "", err
	}

	scopes := make([]string, 0, len(user.Scopes))
	for _, scope := range user.Scopes {
		scopes = append(scopes, scope.Name)
	}

	accessToken, err := s.generateAccessToken(userId, scopes)
	if err != nil {
		s.logger.Error("failed to generate new access token", zap.Error(err))
		return "", err
	}

	s.logger.Info("access token refreshed successfully")
	return accessToken, nil
}

func (s *authService) generateAccessToken(userId string, scope []string) (string, error) {
	claims := jwt.MapClaims{
		"sub":   userId,
		"scope": scope,
		"exp":   time.Now().Add(time.Minute * 15).Unix(),
		"iat":   time.Now().Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedAccessToken, err := accessToken.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return signedAccessToken, nil
}

func (s *authService) generateRefreshToken() (string, error) {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	refreshToken := base64.URLEncoding.EncodeToString(b)
	return refreshToken, nil
}
