package repository

import (
	"context"
	"encoding/json" // New import for JSON serialization
	"errors"
	"fmt"
	"time" // New import for cache expiration

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/shenikar/geo_broadcasting_system/internal/service"
)

type IncidentRepository struct {
	db          *pgxpool.Pool
	redisClient *redis.Client
}

func NewIncidentRepository(db *pgxpool.Pool, redisClient *redis.Client) service.IncidentRepository {
	return &IncidentRepository{
		db:          db,
		redisClient: redisClient,
	}
}

// Create создает новую запись об инциденте в бд
func (r *IncidentRepository) Create(ctx context.Context, incident *models.Incident) error {
	query := `
		INSERT INTO incidents (name, description, location, radius_meters, status)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326), $5, $6) RETURNING id, created_at, updated_at;	
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

// GetByID возвращает инцидент по его UUID
func (r *IncidentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	incident := &models.Incident{}
	query := `
		SELECT 
			id,
			name,
			description,
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters,
			status,
			created_at,
			updated_at
		FROM incidents
		WHERE id = $1;
	`
	err := r.db.QueryRow(ctx, query, id).Scan(
		&incident.ID,
		&incident.Name,
		&incident.Description,
		&incident.Latitude,
		&incident.Longitude,
		&incident.RadiusMeters,
		&incident.Status,
		&incident.CreatedAt,
		&incident.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("incident with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get incident by id: %w", err)
	}
	return incident, nil
}

func (r *IncidentRepository) Update(ctx context.Context, incident *models.Incident) error {
	query := `
		UPDATE incidents SET 
			name = $1,
			description = $2,
			location = ST_SetSRID(ST_MakePoint($3, $4), 4326),
			radius_meters = $5,
			status = $6,
			updated_at = NOW()
		WHERE id = $7;
		`
	cmdTag, err := r.db.Exec(ctx, query,
		incident.Name,
		incident.Description,
		incident.Longitude,
		incident.Latitude,
		incident.RadiusMeters,
		incident.Status,
		incident.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update incident: %w", err)
	}

	// Проверка, была ли хоть обновление одной строки, если RowsAffected() == 0, значит инцидента с таким id не существует
	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("incident with id %s not found for update", incident.ID)
	}
	return nil
}

// Delete(деактивация) устанавливает статус 'inactive' для инцидента
func (r *IncidentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE incidents SET
			status = 'inactive',
			updated_at = NOW()
		WHERE id = $1;
	`
	cmdTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate incident: %w", err)
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("incident with id %s not found for deactivate", id)
	}
	return nil
}

// List возвращает список инцидентов с пагинацией
func (r *IncidentRepository) ListIncidents(ctx context.Context, page, pageSize int) ([]*models.Incident, error) {
	// рассчитываем смещение
	offset := (page - 1) * pageSize

	query := `
		SELECT 
			id,
			name,
			description,
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters,
			status,
			created_at,
			updated_at
		FROM incidents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2;
	`
	rows, err := r.db.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list incidents: %w", err)
	}
	defer rows.Close()

	incidents := make([]*models.Incident, 0)
	for rows.Next() {
		incident := &models.Incident{}
		err := rows.Scan(
			&incident.ID,
			&incident.Name,
			&incident.Description,
			&incident.Latitude,
			&incident.Longitude,
			&incident.RadiusMeters,
			&incident.Status,
			&incident.CreatedAt,
			&incident.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row: %w", err)
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error list iteration: %w", err)
	}
	return incidents, nil
}

// FindActiveByLocation находит активные инциденты, в радиус которых попадает точка
func (r *IncidentRepository) FindActiveLocation(ctx context.Context, lat, lon float64) ([]*models.Incident, error) {
	query := `
		SELECT 
			id,
			name,
			description,
			ST_Y(location::geometry) as latitude,
			ST_X(location::geometry) as longitude,
			radius_meters,
			status,
			created_at,
			updated_at
		FROM incidents
		WHERE
			status = 'active'
			AND ST_DWithin(
				location,
				ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography,
				radius_meters
			);
		`
	rows, err := r.db.Query(ctx, query, lon, lat)
	if err != nil {
		return nil, fmt.Errorf("failed to find active incidents by location: %w", err)
	}
	defer rows.Close()
	incidents := make([]*models.Incident, 0)
	for rows.Next() {
		incident := &models.Incident{}
		err := rows.Scan(
			&incident.ID,
			&incident.Name,
			&incident.Description,
			&incident.Latitude,
			&incident.Longitude,
			&incident.RadiusMeters,
			&incident.Status,
			&incident.CreatedAt,
			&incident.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan incident row in FindActiveLocation: %w", err)
		}
		incidents = append(incidents, incident)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error list iteration in FindActiveLocation: %w", err)
	}
	return incidents, nil
}

// GetLocationCheckStats возвращает количество уникальных пользователей, проверивших геолокацию
func (r *IncidentRepository) GetLocationCheckStats(ctx context.Context, minutes int) (int, error) {
	query := `
		SELECT COUNT(DISTINCT user_id)
		FROM location_checks
		WHERE checked_at >= NOW() - ($1 * INTERVAL '1 minute');
	`
	var count int
	err := r.db.QueryRow(ctx, query, minutes).Scan(&count)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get location check stats: %w", err)
	}
	return count, nil
}

// SaveLocationCheck сохраняет запись о проверке местоположения в бд
func (r *IncidentRepository) SaveLocationCheck(ctx context.Context, check *models.LocationCheck) error {
	query := `
		INSERT INTO location_checks (user_id, location, is_dangerous)
		VALUES ($1, ST_SetSRID(ST_MakePoint($2, $3), 4326), $4) RETURNING id, checked_at;
	`
	err := r.db.QueryRow(ctx, query,
		check.UserID,
		check.Longitude,
		check.Latitude,
		check.IsDangerous,
	).Scan(&check.ID, &check.CheckedAt)
	if err != nil {
		return fmt.Errorf("failed to save location check: %w", err)
	}
	return nil
}

// GetIncidentFromCache пытается получить инцидент из Redis
func (r *IncidentRepository) GetIncidentFromCache(ctx context.Context, id uuid.UUID) (*models.Incident, error) {
	key := fmt.Sprintf("incident:%s", id.String())
	val, err := r.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get incident from cache: %w", err)
	}

	incident := &models.Incident{}
	if err := json.Unmarshal(val, incident); err != nil {
		return nil, fmt.Errorf("failed to unmarshal incident from cache: %w", err)
	}
	return incident, nil
}

// SetIncidentCache сохраняет инцидент в Redis
func (r *IncidentRepository) SetIncidentCache(ctx context.Context, incident *models.Incident) error {
	key := fmt.Sprintf("incident:%s", incident.ID.String())
	val, err := json.Marshal(incident)
	if err != nil {
		return fmt.Errorf("failed to marshal incident for cache: %w", err)
	}
	// Устанавливаем срок жизни кэша, например, 5 минут
	if err := r.redisClient.Set(ctx, key, val, 5*time.Minute).Err(); err != nil {
		return fmt.Errorf("failed to set incident in cache: %w", err)
	}
	return nil
}

// InvalidateIncidentCache удаляет инцидент из Redis кэша
func (r *IncidentRepository) InvalidateIncidentCache(ctx context.Context, id uuid.UUID) error {
	key := fmt.Sprintf("incident:%s", id.String())
	if err := r.redisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to invalidate incident cache: %w", err)
	}
	return nil
}
