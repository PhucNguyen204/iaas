package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/PhucNguyen204/vcs-authentication-service/pkg/env"
)

type IJWTMiddleware interface {
	CheckBearerAuth() gin.HandlerFunc
}

type jwtMiddleware struct {
	jwtSecret []byte
}

func NewJWTMiddleware(env env.AuthEnv) IJWTMiddleware {
	return &jwtMiddleware{
		jwtSecret: []byte(env.JWTSecret),
	}
}

func (m *jwtMiddleware) CheckBearerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
			c.Abort()
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		jwtToken, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			return m.jwtSecret, nil
		})
		if err != nil || !jwtToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims, ok := jwtToken.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid claims"})
			c.Abort()
			return
		}

		if sub, ok := claims["sub"].(string); ok {
			c.Set("userId", sub)
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Insufficient userId"})
			return
		}
		c.Next()
	}
}
