package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/shenikar/geo_broadcasting_system/internal/webhook"
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
	GetLocationCheckStats(ctx context.Context, minutes int) (int, error)
	SaveLocationCheck(ctx context.Context, check *models.LocationCheck) error

	// Методы кэширования
	GetIncidentFromCache(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	SetIncidentCache(ctx context.Context, incident *models.Incident) error
	InvalidateIncidentCache(ctx context.Context, id uuid.UUID) error
}

// IncidentService определяет контрак для бизнес-логики управления инцидентами
type IncidentService interface {
	CreateIncident(ctx context.Context, incident *models.Incident) error
	GetIncident(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	UpdateIncident(ctx context.Context, incident *models.Incident) error
	DeactivateIncident(ctx context.Context, id uuid.UUID) error
	ListIncidents(ctx context.Context, page, pageSize int) ([]*models.Incident, error)
	CheckLocation(ctx context.Context, userID string, lat, lon float64) ([]*models.Incident, error)
	GetStats(ctx context.Context) (int, error)
}

type incidentService struct {
	repo             IncidentRepository
	logger           *logrus.Logger
	cfg              *config.Config
	webhookPublisher webhook.WebhookPublisher
}

func NewIncidentService(repo IncidentRepository, logger *logrus.Logger, cfg *config.Config, publisher webhook.WebhookPublisher) IncidentService {
	return &incidentService{
		repo:             repo,
		logger:           logger,
		cfg:              cfg,
		webhookPublisher: publisher,
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
	// Инвалидируем кэш для этого инцидента (на всякий случай, хотя его еще нет)
	if err := s.repo.InvalidateIncidentCache(ctx, incident.ID); err != nil {
		log.WithError(err).Warn("Failed to invalidate incident cache after creation")
	}
	// TODO: Инвалидировать кеш для списка инцидентов, если он будет реализован
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

	// 1. Попытаться получить из кэша
	incident, err := s.repo.GetIncidentFromCache(ctx, id)
	if err != nil {
		log.WithError(err).Warn("Failed to get incident from cache")
		// Продолжаем, пытаясь получить из БД
	}
	if incident != nil {
		log.Info("Incident found in cache")
		return incident, nil
	}

	log.Info("Incident not found in cache, fetching from DB")
	// 2. Если не в кэше, получить из БД
	incident, err = s.repo.GetByID(ctx, id)
	if err != nil {
		log.WithError(err).Error("Failed to get incident from repository")
		return nil, fmt.Errorf("service: could not get incident: %w", err)
	}

	// 3. Сохранить в кэш
	if err := s.repo.SetIncidentCache(ctx, incident); err != nil {
		log.WithError(err).Warn("Failed to set incident in cache")
		// Это не критическая ошибка, продолжаем
	}

	log.Info("Incident fetched successfully from DB and cached")
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
		return fmt.Errorf("service: incident with id %s not found for update: %w", incident.ID, err)
	}

	existing.Name = incident.Name
	existing.Description = incident.Description
	existing.Latitude = incident.Latitude
	existing.Longitude = incident.Longitude
	existing.RadiusMeters = incident.RadiusMeters
	existing.Status = incident.Status

	if err := s.repo.Update(ctx, existing); err != nil {
		log.WithError(err).Error("Failed to update incident in repository")
		return fmt.Errorf("service: could not update incident: %w", err)
	}
	log.Info("Incident updated successfully")

	// Инвалидируем кэш для обновленного инцидента
	if err := s.repo.InvalidateIncidentCache(ctx, incident.ID); err != nil {
		log.WithError(err).Warn("Failed to invalidate incident cache after update")
	}
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
		return fmt.Errorf("service: incident with id %s not found for deactivate: %w", id, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		log.WithError(err).Error("Failed to deactivate incident in repository")
		return fmt.Errorf("service: could not deactivate incident: %w", err)
	}

	log.Info("Incident deactivated successfully")
	// Инвалидируем кэш для деактивированного инцидента
	if err := s.repo.InvalidateIncidentCache(ctx, id); err != nil {
		log.WithError(err).Warn("Failed to invalidate incident cache after deactivation")
	}
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

// CheckLocation находит активные инциденты и публикует вебхук при наличии опасности
func (s *incidentService) CheckLocation(ctx context.Context, userID string, lat, lon float64) ([]*models.Incident, error) {
	log := s.logger.WithFields(logrus.Fields{
		"service": "incident",
		"method":  "CheckLocation",
		"user_id": userID,
	})
	log.Info("Checking user location")

	activeIncident, err := s.repo.FindActiveLocation(ctx, lat, lon)
	if err != nil {
		log.WithError(err).Error("Failed to find active incidents by location")
		return nil, fmt.Errorf("service: failed to find active incidents: %w", err)
	}
	isDanger := len(activeIncident) > 0

	// Сохраняем факт проверки местоположения
	locationCheck := &models.LocationCheck{
		UserID:      userID,
		Latitude:    lat,
		Longitude:   lon,
		IsDangerous: isDanger,
	}
	if err := s.repo.SaveLocationCheck(ctx, locationCheck); err != nil {
		log.WithError(err).Error("Failed to save location check to repository")
		// Это не критическая ошибка, продолжаем выполнение
	}

	log.WithField("is_danger", isDanger).Info("Location check completed")

	// Публикуем вебхук, если обнаружена опасность
	if isDanger {
		webhookEvent := webhook.WebhookEvent{
			UserID:      userID,
			Latitude:    lat,
			Longitude:   lon,
			IsDangerous: isDanger,
			Timestamp:   time.Now(),
			Incidents:   activeIncident,
		}
		if err := s.webhookPublisher.Publish(ctx, webhookEvent); err != nil {
			log.WithError(err).Error("Failed to publish webhook event")
			// Это не критическая ошибка, продолжаем выполнение
		} else {
			log.Info("Webhook event published successfully")
		}
	}

	return activeIncident, nil
}

// GetStats возвращает количество уникальных пользователей, проверивших геолокацию
func (s *incidentService) GetStats(ctx context.Context) (int, error) {
	log := s.logger.WithFields(logrus.Fields{
		"service": "incident",
		"method":  "GetStats",
	})
	log.Info("Getting location check stats")

	userCount, err := s.repo.GetLocationCheckStats(ctx, s.cfg.StatsTimeWindowMinutes)
	if err != nil {
		log.WithError(err).Error("Failed to get location check stats from repository")
		return 0, fmt.Errorf("service: failed to get location check stats: %w", err)
	}

	log.WithField("user_count", userCount).Info("Location check stats retrieved successfully")
	return userCount, nil
}
