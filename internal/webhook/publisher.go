package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
)

const (
	webhookQueueKey = "webhook_events"
)

// WebhookEvent - структура для данных вебхука
type WebhookEvent struct {
	UserID      string             `json:"user_id"`
	Latitude    float64            `json:"latitude"`
	Longitude   float64            `json:"longitude"`
	IsDangerous bool               `json:"is_dangerous"`
	Timestamp   time.Time          `json:"timestamp"`
	Incidents   []*models.Incident `json:"incidents,omitempty"` // Список инцидентов, если пользователь в опасной зоне
}

// WebhookPublisher - интерфейс для публикации вебхуков
type WebhookPublisher interface {
	Publish(ctx context.Context, event WebhookEvent) error
}

// RedisWebhookPublisher - реализация WebhookPublisher, использующая Redis
type RedisWebhookPublisher struct {
	redisClient *redis.Client
}

// NewRedisWebhookPublisher создает новый RedisWebhookPublisher
func NewRedisWebhookPublisher(client *redis.Client) *RedisWebhookPublisher {
	return &RedisWebhookPublisher{
		redisClient: client,
	}
}

// Publish публикует событие вебхука в очередь Redis
func (p *RedisWebhookPublisher) Publish(ctx context.Context, event WebhookEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook event: %w", err)
	}

	// Используем LPUSH для добавления события в левую часть списка (очереди)
	if err := p.redisClient.LPush(ctx, webhookQueueKey, payload).Err(); err != nil {
		return fmt.Errorf("failed to publish webhook event to Redis: %w", err)
	}
	return nil
}
