package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
)

// NewPostgresDB создает новый пул соединений PostgreSQL
func NewPostgresDB(ctx context.Context, appCfg *config.Config) (*pgxpool.Pool, error) {
	cfgPool, err := pgxpool.ParseConfig(appCfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("ошибка при разборе конфигурации postgres: %w", err)
	}

	dbpool, err := pgxpool.NewWithConfig(ctx, cfgPool)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать пул соединений: %w", err)
	}

	// Проверяем соединение с базой данных
	err = dbpool.Ping(ctx)
	if err != nil {
		dbpool.Close()
		return nil, fmt.Errorf("не удалось выполнить ping к postgres: %w", err)
	}

	return dbpool, nil
}
