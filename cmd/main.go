package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/shenikar/geo_broadcasting_system/internal/config"
	v1 "github.com/shenikar/geo_broadcasting_system/internal/handler/http/v1"
	"github.com/shenikar/geo_broadcasting_system/internal/repository"
	"github.com/shenikar/geo_broadcasting_system/internal/service"
	"github.com/shenikar/geo_broadcasting_system/internal/webhook"
	"github.com/shenikar/geo_broadcasting_system/pkg/logger"
	"github.com/shenikar/geo_broadcasting_system/pkg/postgres"
	redisclient "github.com/shenikar/geo_broadcasting_system/pkg/redis"
	"github.com/sirupsen/logrus"

	_ "github.com/shenikar/geo_broadcasting_system/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Geo Broadcasting System API
// @version 1.0
// @description This is a Geo Broadcasting System API server.
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
func runMigrations(cfg *config.Config, log *logrus.Logger) error {
	log.Info("Running database migrations...")

	migrationURL := cfg.DatabaseURL
	if !strings.HasPrefix(migrationURL, "pgx5://") {
		migrationURL = strings.Replace(migrationURL, "postgres://", "pgx5://", 1)
	}

	m, err := migrate.New(
		"file://migrations",
		migrationURL,
	)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	log.Info("Database migrations applied successfully")
	return nil
}

func main() {
	// Загрузка конфигурации
	cfg, err := config.LoadConfig()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	// Инициализация логгера
	log := logger.New(cfg.LogLevel)

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск миграций
	if err := runMigrations(cfg, log); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	// Подключение к PostgreSQL
	dbpool, err := postgres.NewPostgresDB(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer dbpool.Close()
	log.Info("Successfully connected to PostgreSQL")

	// Инициализация Redis клиента
	redisClient, err := redisclient.NewRedisClient(ctx, cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Info("Successfully connected to Redis")

	// Инициализация издателя вебхуков
	webhookPublisher := webhook.NewRedisWebhookPublisher(redisClient)

	// Инициализация и запуск воркера вебхуков
	webhookWorker := webhook.NewWebhookWorker(redisClient, log, cfg)
	webhookWorker.Start(ctx)
	// Инициализация репозиториев
	incidentRepo := repository.NewIncidentRepository(dbpool, redisClient)

	// Инициализация сервисов
	incidentService := service.NewIncidentService(incidentRepo, log, cfg, webhookPublisher)

	// Инициализация хэндлеров
	handler := v1.NewHandler(incidentService, log, cfg)

	// Настройка Gin роутера
	router := gin.Default()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	// Добавление маршрута для Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Запуск HTTP-сервера
	serverAddr := fmt.Sprintf(":%s", cfg.HTTPPort)

	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Запуск сервера в горутине
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
	}()
	log.Infof("HTTP server started on port %s", cfg.HTTPPort)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("Received shutdown signal, shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Info("Server gracefully stopped")
}
