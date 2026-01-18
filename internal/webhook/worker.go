package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/sirupsen/logrus"
)

// WebhookWorker - структура для обработки и отправки вебхуков
type WebhookWorker struct {
	redisClient *redis.Client
	logger      *logrus.Logger
	cfg         *config.Config
	httpClient  *http.Client
}

// NewWebhookWorker создает новый WebhookWorker
func NewWebhookWorker(redisClient *redis.Client, logger *logrus.Logger, cfg *config.Config) *WebhookWorker {
	return &WebhookWorker{
		redisClient: redisClient,
		logger:      logger,
		cfg:         cfg,
		httpClient: &http.Client{
			Timeout: cfg.WebhookTimeout,
		},
	}
}

// Start запускает горутину для обработки очереди вебхуков
func (w *WebhookWorker) Start(ctx context.Context) {
	w.logger.Info("Starting webhook worker...")
	go func() {
		for {
			select {
			case <-ctx.Done():
				w.logger.Info("Stopping webhook worker.")
				return
			default:
				// BLPOP - блокирующее извлечение из правой части списка (очереди)
				// 0 означает бесконечное ожидание
				result, err := w.redisClient.BRPop(ctx, 0, webhookQueueKey).Result()
				if err != nil {
					if errors.Is(err, context.Canceled) {
						continue // Контекст отменен, но не ошибка Redis
					}
					w.logger.WithError(err).Error("Failed to pop webhook event from Redis")
					time.Sleep(w.cfg.WebhookTimeout) // Ждем перед повторной попыткой
					continue
				}

				// result[0] - ключ, result[1] - значение
				payload := result[1]
				var event WebhookEvent
				if err := json.Unmarshal([]byte(payload), &event); err != nil {
					w.logger.WithError(err).Error("Failed to unmarshal webhook event from Redis")
					continue
				}

				w.processWebhookEvent(ctx, event, payload)
			}
		}
	}()
}

func (w *WebhookWorker) processWebhookEvent(ctx context.Context, event WebhookEvent, rawPayload string) {
	log := w.logger.WithField("event_user_id", event.UserID).WithField("event_is_dangerous", event.IsDangerous)
	log.Debug("Processing webhook event...")

	if w.cfg.WebhookURL == "" {
		log.Warn("Webhook URL is not configured. Skipping webhook delivery.")
		return
	}

	maxRetries := w.cfg.WebhookMaxRetries
	baseDelay := w.cfg.WebhookBaseDelay

	for i := 0; i < maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", w.cfg.WebhookURL, bytes.NewBufferString(rawPayload))
		if err != nil {
			log.WithError(err).Errorf("Failed to create webhook request for event. Retries left: %d", maxRetries-1-i)
			continue
		}

		req.Header.Set("Content-Type", "application/json")

		// Добавляем HMAC подпись, если WEBHOOK_SECRET задан
		if w.cfg.WebhookSecret != "" {
			signature := generateHMACSHA256(rawPayload, w.cfg.WebhookSecret)
			req.Header.Set("X-Webhook-Signature", signature)
		}

		resp, err := w.httpClient.Do(req)
		if err != nil {
			log.WithError(err).Warnf("Failed to send webhook for event. Retrying in %v. Retries left: %d", baseDelay, maxRetries-1-i)
			time.Sleep(baseDelay)
			baseDelay *= 2 // Экспоненциальная задержка
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Info("Webhook delivered successfully.")
			return
		} else {
			log.Warnf("Webhook delivery failed with status code %d. Retrying in %v. Retries left: %d", resp.StatusCode, baseDelay, maxRetries-1-i)
			time.Sleep(baseDelay)
			baseDelay *= 2 // Экспоненциальная задержка
		}
	}

	log.Errorf("Failed to deliver webhook for event after %d retries.", maxRetries)
}

// generateHMACSHA256 генерирует HMAC-SHA256 подпись для данных
func generateHMACSHA256(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
