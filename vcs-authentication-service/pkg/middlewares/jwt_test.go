package middlewares

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/PhucNguyen204/vcs-authentication-service/pkg/env"
)

type JWTMiddlewareSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	jwtMiddleware IJWTMiddleware
	router        *gin.Engine
	testSecret    string
	ctx           context.Context
}

func (s *JWTMiddlewareSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.testSecret = "test-secret-key"
	s.ctx = context.Background()

	authEnv := env.AuthEnv{
		JWTSecret: s.testSecret,
	}

	s.jwtMiddleware = NewJWTMiddleware(authEnv)

	gin.SetMode(gin.TestMode)
	s.router = gin.New()
}

func (s *JWTMiddlewareSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestJWTMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(JWTMiddlewareSuite))
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuth() {
	claims := jwt.MapClaims{
		"sub":   "123",
		"name":  "testuser",
		"scope": []interface{}{"read", "write"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.testSecret))
	s.Require().NoError(err)

	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		userId, exists := c.Get("userId")
		s.True(exists)
		s.Equal("123", userId)
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("success", response["message"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthMissingAuthHeader() {
	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Missing or invalid token", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthInvalidAuthHeader() {
	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Missing or invalid token", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthInvalidToken() {
	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Invalid token", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthExpiredToken() {
	claims := jwt.MapClaims{
		"sub":   "123",
		"name":  "testuser",
		"scope": []interface{}{"read", "write"},
		"exp":   time.Now().Add(-time.Hour).Unix(),
		"iat":   time.Now().Add(-time.Hour * 2).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.testSecret))
	s.Require().NoError(err)

	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Invalid token", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthWrongSecret() {
	claims := jwt.MapClaims{
		"sub":   "123",
		"name":  "testuser",
		"scope": []interface{}{"read", "write"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte("wrong-secret"))
	s.Require().NoError(err)

	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Invalid token", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthMissingUserId() {
	claims := jwt.MapClaims{
		"name":  "testuser",
		"scope": []interface{}{"read", "write"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.testSecret))
	s.Require().NoError(err)

	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Insufficient userId", response["error"])
}

func (s *JWTMiddlewareSuite) TestCheckBearerAuthInvalidUserIdType() {
	claims := jwt.MapClaims{
		"sub":   123,
		"name":  "testuser",
		"scope": []interface{}{"read", "write"},
		"exp":   time.Now().Add(time.Hour).Unix(),
		"iat":   time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.testSecret))
	s.Require().NoError(err)

	s.router.GET("/test", s.jwtMiddleware.CheckBearerAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	w := httptest.NewRecorder()

	s.router.ServeHTTP(w, req)
	s.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	s.NoError(err)
	s.Equal("Insufficient userId", response["error"])
}
