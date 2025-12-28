package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/shenikar/geo_advertising_system/internal/models"
)

// IncidentRepository определяет контракт для работы с бд инцидентов
type IncidentRepository interface {
	Create(ctx context.Context, incident *models.Incident) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error)
	Update(ctx context.Context, incident *models.Incident) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, pageSize int) ([]*models.Incident, error)
	FindActiveByLocation(ctx context.Context, lat, lon float64) ([]*models.Incident, error)
}
