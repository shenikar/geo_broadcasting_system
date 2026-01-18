package v1

import (
	"time"

	"github.com/google/uuid"
)

// CreateIncidentRequest DTO для создания инцидента
// @Description DTO для создания инцидента
type CreateIncidentRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255"`
	Description  string  `json:"description,omitempty"`
	Latitude     float64 `json:"latitude" validate:"required,latitude"`
	Longitude    float64 `json:"longitude" validate:"required,longitude"`
	RadiusMeters int     `json:"radius_meters" validate:"required,gt=0"`
}

// UpdateIncidentRequest DTO для обновления инцидента
// @Description DTO для обновления инцидента
type UpdateIncidentRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255"`
	Description  string  `json:"description,omitempty"`
	Latitude     float64 `json:"latitude" validate:"required,latitude"`
	Longitude    float64 `json:"longitude" validate:"required,longitude"`
	RadiusMeters int     `json:"radius_meters" validate:"required,gt=0"`
	Status       string  `json:"status" validate:"required,oneof=active inactive"`
}

// IncidentResponse DTO для ответа с информацией об инциденте
// @Description DTO для ответа с информацией об инциденте
type IncidentResponse struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	RadiusMeters int       `json:"radius_meters"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// LocationCheckRequest DTO для проверки координат
// @Description DTO для проверки координат
type LocationCheckRequest struct {
	UserID    string  `json:"user_id" validate:"required"`
	Latitude  float64 `json:"latitude" validate:"required,latitude"`
	Longitude float64 `json:"longitude" validate:"required,longitude"`
}

// StatsResponse DTO для ответа со статистикой
// @Description DTO для ответа со статистикой
type StatsResponse struct {
	UserCount int `json:"user_count"`
}
