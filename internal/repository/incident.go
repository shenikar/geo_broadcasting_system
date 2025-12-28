package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/shenikar/geo_broadcasting_system/internal/service"
)

type IncidentRepository struct {
	db *pgxpool.Pool
}

func NewIncidentRepository(db *pgxpool.Pool) service.IncidentRepository {
	return &IncidentRepository{
		db: db,
	}
}

// Create создает новую запись об инциденте в бд
func (r *IncidentRepository) Create(ctx context.Context, incident *models.Incident) error {
	query := `
		INSERT INTO incidents (name, description, location, radius_meters, status)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326), $5, $6) RETURING id, created_at, updated_at;	
	`
	err := r.db.QueryRow(ctx, query,
		incident.Name,
		incident.Description,
		incident.Longitude,
		incident.Latitude,
		incident.RadiusMeters,
		incident.Status,
	).Scan(&incident.ID, &incident.CreatedAt, &incident.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create incident: %w", err)
	}
	return nil
}
