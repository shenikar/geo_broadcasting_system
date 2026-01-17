package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/service"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	incidentService service.IncidentService
	logger          *logrus.Logger
	validate        *validator.Validate
}

func NewHandler(incidentService service.IncidentService, logger *logrus.Logger) *Handler {
	return &Handler{
		incidentService: incidentService,
		logger:          logger,
		validate:        validator.New(),
	}
}

func (h *Handler) createIncident(c *gin.Context) {
	var input CreateIncidentRequest
	log := h.logger.WithField("method", "createIncident")

	if err := c.ShouldBindJSON(&input); err != nil {
		log.WithError(err).Warn("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.validate.Struct(input); err != nil {
		log.WithError(err).Warn("Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model := DTOToIncidentModel(input)
	if err := h.incidentService.CreateIncident(c.Request.Context(), model); err != nil {
		log.WithError(err).Error("Failed to create incident in service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}
	c.JSON(http.StatusCreated, ModelToIncidentResponse(model))
}

func (h *Handler) listIncidents(c *gin.Context) {
	log := h.logger.WithField("method", "listIncidents")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	incidents, err := h.incidentService.ListIncidents(c.Request.Context(), page, pageSize)
	if err != nil {
		log.WithError(err).Error("Failed to list incident from service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, ModelsToIncidentResponses(incidents))
}

func (h *Handler) getIncident(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident ID"})
		return
	}
	log := h.logger.WithField("method", "getIncident").WithField("id", id)

	incident, err := h.incidentService.GetIncident(c.Request.Context(), id)
	if err != nil {
		log.WithError(err).Warn("Failed to get incident from service")
		c.JSON(http.StatusNotFound, gin.H{"error": "incident not found"})
		return
	}
	c.JSON(http.StatusOK, ModelToIncidentResponse(incident))
}

func (h *Handler) updateIncident(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident ID"})
		return
	}
	log := h.logger.WithField("method", "updateIncident").WithField("id", id)

	var input UpdateIncidentRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		log.WithError(err).Warn("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.validate.Struct(input); err != nil {
		log.WithError(err).Warn("Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	model := DTOToIncidentModel(input)
	model.ID = id

	if err := h.incidentService.UpdateIncident(c.Request.Context(), model); err != nil {
		log.WithError(err).Error("Failed to update incident in service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update incident in service"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) deleteIncident(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid incident ID"})
		return
	}
	log := h.logger.WithField("method", "deleteIncident").WithField("id", id)

	if err := h.incidentService.DeactivateIncident(c.Request.Context(), id); err != nil {
		log.WithError(err).Error("Failed to deactivate incident in service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate incident"})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) checkLocation(c *gin.Context) {
	var input LocationCheckRequest
	log := h.logger.WithField("method", "checkLocation")

	if err := c.ShouldBindJSON(&input); err != nil {
		log.WithError(err).Warn("Failed to bind JSON")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.validate.Struct(input); err != nil {
		log.WithError(err).Warn("Validation failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	incidents, err := h.incidentService.CheckLocation(c.Request.Context(), input.UserID, input.Latitude, input.Longitude)
	if err != nil {
		log.WithError(err).Error("Failed to check location in service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, ModelsToIncidentResponses(incidents))
}

func (h *Handler) getStats(c *gin.Context) {
	log := h.logger.WithField("method", "getStats")

	userCount, err := h.incidentService.GetStats(c.Request.Context())
	if err != nil {
		log.WithError(err).Error("Failed to get stats from service")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, StatsResponse{UserCount: userCount})
}

func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
