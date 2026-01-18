package v1

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/shenikar/geo_broadcasting_system/internal/service"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	incidentService service.IncidentService
	logger          *logrus.Logger
	validate        *validator.Validate
	cfg             *config.Config
}

func NewHandler(incidentService service.IncidentService, logger *logrus.Logger, cfg *config.Config) *Handler {
	return &Handler{
		incidentService: incidentService,
		logger:          logger,
		validate:        validator.New(),
		cfg:             cfg,
	}
}

// @Summary Create a new incident
// @Description Create a new incident in the system. Requires API key.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param incident body CreateIncidentRequest true "Incident creation request"
// @Success 201 {object} IncidentResponse
// @Failure 400 {object} map[string]string "Invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /incidents [post]
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

// @Summary Get a list of incidents
// @Description Get a paginated list of all incidents. Requires API key.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Number of items per page" default(10)
// @Success 200 {array} IncidentResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /incidents [get]
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

// @Summary Get incident by ID
// @Description Get a single incident by its ID. Requires API key.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Incident ID"
// @Success 200 {object} IncidentResponse
// @Failure 400 {object} map[string]string "Invalid incident ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Incident not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /incidents/{id} [get]
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

// @Summary Update an existing incident
// @Description Update an existing incident by ID. Requires API key.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Incident ID"
// @Param incident body UpdateIncidentRequest true "Incident update request"
// @Success 200 "OK"
// @Failure 400 {object} map[string]string "Invalid incident ID or request body"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /incidents/{id} [put]
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

// @Summary Deactivate an incident
// @Description Deactivate an incident by its ID. This marks the incident as inactive. Requires API key.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Incident ID"
// @Success 204 "No Content"
// @Failure 400 {object} map[string]string "Invalid incident ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /incidents/{id} [delete]
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

// @Summary Check location for incidents
// @Description Check if there are any active incidents at a given location for a user. Requires API key.
// @Tags Location
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param location body LocationCheckRequest true "Location check request"
// @Success 200 {array} IncidentResponse
// @Failure 400 {object} map[string]string "Invalid request body or validation error"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /location/check [post]
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

// @Summary Get user statistics
// @Description Get the total count of active users. Requires API key.
// @Tags Admin
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} StatsResponse
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /stats [get]
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

// @Summary Get application health status
// @Description Get health status of the application
// @Tags System
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string "Status OK"
// @Router /system/health [get]
func (h *Handler) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
