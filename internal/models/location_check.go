package models

import (
	"time"
)

// LocationCheck представляет запись о проверке местоположения пользователя
type LocationCheck struct {
	ID          int64     `json:"id"`
	UserID      string    `json:"user_id"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	IsDangerous bool      `json:"is_dangerous"`
	CheckedAt   time.Time `json:"checked_at"`
}
