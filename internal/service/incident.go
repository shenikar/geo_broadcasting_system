package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/sirupsen/logrus"
)

// IncidentRepository определяет контракт для работы с бд инцидентов
type IncidentRepository interface {
	Create(ctx context.Context, incident *models.Incident) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	Update(ctx context.Context, incident *models.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListIncidents(ctx context.Context, page, pageSize int) ([]*models.Incident, error)
	FindActiveLocation(ctx context.Context, lat, lon float64) ([]*models.Incident, error)
}

// IncidentService определяет контрак для бизнес-логики управления инцидентами
type IncidentService interface {
	CreateIncident(ctx context.Context, incident *models.Incident) error
	GetIncident(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	UpdateIncident(ctx context.Context, incident *models.Incident) error
	DeactivateIncident(ctx context.Context, id uuid.UUID) error
	ListIncidents(ctx context.Context, page, pageSize int) ([]*models.Incident, error)
	CheckLocation(ctx context.Context, userID string, lat, lon float64) ([]*models.Incident, error)
}

type incidentService struct {
	repo   IncidentRepository
	logger *logrus.Logger
}

func NewIncidentService(repo IncidentRepository, logger *logrus.Logger) IncidentService {
	return &incidentService{
		repo:   repo,
		logger: logger,
	}
}

// CreateIncident создает инцидент
func (s *incidentService) CreateIncident(ctx context.Context, incident *models.Incident) error {
	log := s.logger.WithFields(logrus.Fields{
		"service": "incident",
		"method":  "CreateIncident",
		"name":    incident.Name,
	})
	log.Info("Attempting to create a new incident")

	incident.Status = "active"
	if err := s.repo.Create(ctx, incident); err != nil {
		log.WithError(err).Error("Failed to create incident in repository")
		return fmt.Errorf("service: could not create incident: %w", err)
	}

	log.WithField("incident_id", incident.ID).Info("Incident created successfully")
	// TODO: Инвалидировать кеш для списка инцидентов
	return nil
}

// GetIncident получает инцидент по ID
func (s *incidentService) GetIncident(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	log := s.logger.WithFields(logrus.Fields{
		"service":     "incident",
		"method":      "GetIncident",
		"incident_id": id,
	})
	log.Info("Fetching incident by ID")
	// TODO: Добавить логику кеширования
	incident, err := s.repo.GetByID(ctx, id)
	if err != nil {
		log.WithError(err).Error("Failed to get incident in repository")
		return nil, fmt.Errorf("service: not get incident: %w", err)
	}

	log.Info("Incident fetched successfully")
	return incident, nil
}

// UpdateIncident обновляет существующий инцидент.
func (s *incidentService) UpdateIncident(ctx context.Context, incident *models.Incident) error {
	log := s.logger.WithFields(logrus.Fields{
		"service":     "incident",
		"method":      "UpdateIncident",
		"incident_id": incident.ID,
	})
	log.Info("Attempting to update a new incident")
	existing, err := s.repo.GetByID(ctx, incident.ID)
	if err != nil {
		log.WithError(err).Warn("Attempted to update a non-existent incident")
		return fmt.Errorf("service: incident with id %snot found for update: %w", incident.ID, err)
	}

	existing.Name = incident.Name
	existing.Description = incident.Description
	existing.Latitude = incident.Latitude
	existing.Longitude = incident.Longitude
	existing.RadiusMeters = incident.RadiusMeters
	existing.Status = incident.Description

	if err := s.repo.Update(ctx, existing); err != nil {
		log.WithError(err).Error("Failed to update incident in repository")
		return fmt.Errorf("service: could not update incident: %w", err)
	}
	log.Info("Incident updated successfully")
	return nil
}

// DeactivateIncident дективирует инцидент
func (s *incidentService) DeactivateIncident(ctx context.Context, id uuid.UUID) error {
	log := s.logger.WithFields(logrus.Fields{
		"service":     "incident",
		"method":      "DeactivateIncident",
		"incident_id": id,
	})
	log.Info("Attempting to deactivate incident")

	if _, err := s.repo.GetByID(ctx, id); err != nil {
		log.WithError(err).Warn("Attempted to deactivate a non-existent incident")
		return fmt.Errorf("service: incident with id %snot found for deactivate: %w", id, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.WithError(err).Error("Failed to deactivate incident in repository")
		return fmt.Errorf("service: could not deactivate incident: %w", err)
	}

	log.Info("Incident deactivated successfully")
	return nil

}

// ListIncidents возвращает список инцидентов с пагинацией
func (s *incidentService) ListIncidents(ctx context.Context, page, pageSize int) ([]*models.Incident, error) {
	if page < 1 {
		page = 1
	}

	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	log := s.logger.WithFields(logrus.Fields{
		"service":   "incident",
		"method":    "ListIncidents",
		"page":      page,
		"page_size": pageSize,
	})
	log.Info("Listing incidents")

	incidents, err := s.repo.ListIncidents(ctx, page, pageSize)
	if err != nil {
		log.WithError(err).Error("Failed to list incidents from repository")
		return nil, fmt.Errorf("service: could not list incidents: %w", err)
	}

	log.WithField("count", len(incidents)).Info("Incidents listed successfully")
	return incidents, nil
}

// CheckLocation находит активные инциденты
func (s *incidentService) CheckLocation(ctx context.Context, userID string, lat, lon float64) ([]*models.Incident, error) {
	log := s.logger.WithFields(logrus.Fields{
		"service": "incident",
		"method":  "DeactivateIncident",
		"user_id": userID,
	})
	log.Info("Checking user location")

	activeIncident, err := s.repo.FindActiveLocation(ctx, lat, lon)
	if err != nil {
		log.WithError(err).Error("Failed to find active incidents by location")
		return nil, fmt.Errorf("service: failed to find active incidents: %w", err)
	}
	isDanger := len(activeIncident) > 0
	log.WithField("is_danger", isDanger).Info("Location check completed")

	return activeIncident, nil
}
