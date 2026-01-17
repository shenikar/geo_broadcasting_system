package v1

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes регистрирует все маршруты API v1
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	// Маршруты для управления инцидентами (CRUD), защищенные API ключом
	incidents := api.Group("/incidents")
	incidents.Use(APIKeyAuthMiddleware(h.cfg, h.logger))
	{
		incidents.POST("", h.createIncident)
		incidents.GET("", h.listIncidents)
		incidents.GET("/:id", h.getIncident)
		incidents.PUT("/:id", h.updateIncident)
		incidents.DELETE("/:id", h.deleteIncident)
		incidents.GET("/stats", h.getStats)
	}

	// Маршрут для проверки местоположения (публичный)
	api.POST("/location/check", h.checkLocation)

	// Маршрут Health-check (публичный)
	api.GET("/system/health", h.healthCheck)
}
