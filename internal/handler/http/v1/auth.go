package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/sirupsen/logrus"
)

// APIKeyAuthMiddleware - middleware для аутентификации по API-ключу
func APIKeyAuthMiddleware(cfg *config.Config, log *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			// Проверяем также заголовок Authorization: Bearer
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" && strings.HasPrefix(authHeader, "Bearer ") {
				apiKey = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if apiKey == "" {
			log.Warn("API key missing from request")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "API key required"})
			return
		}

		isValid := false
		for _, key := range cfg.APIKeys {
			if key == apiKey {
				isValid = true
				break
			}
		}

		if !isValid {
			log.Warnf("Invalid API key provided: %s", apiKey)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid API key"})
			return
		}

		c.Next()
	}
}
