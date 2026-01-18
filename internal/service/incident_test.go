package service

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/shenikar/geo_broadcasting_system/internal/service/mocks"
	"github.com/shenikar/geo_broadcasting_system/internal/webhook"
	webhook_mocks "github.com/shenikar/geo_broadcasting_system/internal/webhook/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newTestIncidentService — вспомогательная функция для создания инстанса сервиса с моками.
func newTestIncidentService(t *testing.T) (*incidentService, *mocks.MockIncidentRepository, *webhook_mocks.MockWebhookPublisher) {
	ctrl := gomock.NewController(t)
	repoMock := mocks.NewMockIncidentRepository(ctrl)
	webhookMock := webhook_mocks.NewMockWebhookPublisher(ctrl)

	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{}) // Отключаем вывод логов в тестах

	cfg := &config.Config{
		StatsTimeWindowMinutes: 60,
	}

	service := NewIncidentService(repoMock, logger, cfg, webhookMock)
	return service.(*incidentService), repoMock, webhookMock
}

func TestGetIncident_Success_FromCache(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	expectedIncident := &models.Incident{
		ID:   incidentID,
		Name: "Тестовый инцидент из кеша",
	}

	// Ожидания
	repoMock.EXPECT().
		GetIncidentFromCache(ctx, incidentID).
		Return(expectedIncident, nil).
		Times(1)

	// Действие
	incident, err := service.GetIncident(ctx, incidentID)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, expectedIncident, incident)
}

func TestGetIncident_Success_FromDB(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	expectedIncident := &models.Incident{
		ID:   incidentID,
		Name: "Тестовый инцидент из БД",
	}

	// Ожидания
	// 1. Промах кеша
	repoMock.EXPECT().
		GetIncidentFromCache(ctx, incidentID).
		Return(nil, nil).
		Times(1)

	// 2. Попадание в БД
	repoMock.EXPECT().
		GetByID(ctx, incidentID).
		Return(expectedIncident, nil).
		Times(1)

	// 3. Запись в кеш
	repoMock.EXPECT().
		SetIncidentCache(ctx, expectedIncident).
		Return(nil).
		Times(1)

	// Действие
	incident, err := service.GetIncident(ctx, incidentID)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, expectedIncident, incident)
}

func TestGetIncident_NotFound(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	dbError := fmt.Errorf("не найдено")

	// Ожидания
	// 1. Промах кеша
	repoMock.EXPECT().
		GetIncidentFromCache(ctx, incidentID).
		Return(nil, nil).
		Times(1)

	// 2. Промах в БД
	repoMock.EXPECT().
		GetByID(ctx, incidentID).
		Return(nil, dbError).
		Times(1)

	// Действие
	incident, err := service.GetIncident(ctx, incidentID)

	// Проверки
	require.Error(t, err)
	assert.Nil(t, incident)
	assert.ErrorContains(t, err, "could not get incident")
}

func TestCreateIncident_Success(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentToCreate := &models.Incident{
		Name: "Новый пожар",
	}

	// Ожидания
	repoMock.EXPECT().
		Create(ctx, gomock.Any()).
		// Используем gomock.Any(), так как сервис модифицирует инцидент (ставит ID, статус) перед передачей в репозиторий.
		// Можно использовать и более специфичный матчер.
		DoAndReturn(func(ctx context.Context, inc *models.Incident) error {
			// Симулируем, что БД присвоила ID
			inc.ID = uuid.New()
			inc.Status = "active"
			return nil
		}).Times(1)

	repoMock.EXPECT().
		InvalidateIncidentCache(ctx, gomock.Any()).
		Return(nil).
		Times(1)

	// Действие
	err := service.CreateIncident(ctx, incidentToCreate)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, "active", incidentToCreate.Status)
	assert.NotEqual(t, uuid.Nil, incidentToCreate.ID)
}

func TestUpdateIncident_Success(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	incidentToUpdate := &models.Incident{
		ID:   incidentID,
		Name: "Обновленное имя",
	}
	existingIncident := &models.Incident{
		ID:   incidentID,
		Name: "Старое имя",
	}

	// Ожидания
	repoMock.EXPECT().GetByID(ctx, incidentID).Return(existingIncident, nil).Times(1)
	repoMock.EXPECT().Update(ctx, gomock.Any()).Return(nil).Times(1)
	repoMock.EXPECT().InvalidateIncidentCache(ctx, incidentID).Return(nil).Times(1)

	// Действие
	err := service.UpdateIncident(ctx, incidentToUpdate)

	// Проверки
	require.NoError(t, err)
}

func TestUpdateIncident_NotFound(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	incidentToUpdate := &models.Incident{ID: incidentID}
	repoError := fmt.Errorf("не найдено")

	// Ожидания
	repoMock.EXPECT().GetByID(ctx, incidentID).Return(nil, repoError).Times(1)

	// Действие
	err := service.UpdateIncident(ctx, incidentToUpdate)

	// Проверки
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found for update")
}

func TestDeactivateIncident_Success(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	existingIncident := &models.Incident{ID: incidentID}

	// Ожидания
	repoMock.EXPECT().GetByID(ctx, incidentID).Return(existingIncident, nil).Times(1)
	repoMock.EXPECT().Delete(ctx, incidentID).Return(nil).Times(1)
	repoMock.EXPECT().InvalidateIncidentCache(ctx, incidentID).Return(nil).Times(1)

	// Действие
	err := service.DeactivateIncident(ctx, incidentID)

	// Проверки
	require.NoError(t, err)
}

func TestDeactivateIncident_NotFound(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	incidentID := uuid.New()
	repoError := fmt.Errorf("не найдено")

	// Ожидания
	repoMock.EXPECT().GetByID(ctx, incidentID).Return(nil, repoError).Times(1)

	// Действие
	err := service.DeactivateIncident(ctx, incidentID)

	// Проверки
	require.Error(t, err)
	assert.ErrorContains(t, err, "not found for deactivate")
}

func TestListIncidents_Success(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	page, pageSize := 1, 10
	expectedIncidents := []*models.Incident{
		{ID: uuid.New(), Name: "Инцидент 1"},
		{ID: uuid.New(), Name: "Инцидент 2"},
	}

	// Ожидания
	repoMock.EXPECT().ListIncidents(ctx, page, pageSize).Return(expectedIncidents, nil).Times(1)

	// Действие
	incidents, err := service.ListIncidents(ctx, page, pageSize)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, expectedIncidents, incidents)
}

func TestCheckLocation_Danger(t *testing.T) {
	// Подготовка
	service, repoMock, webhookMock := newTestIncidentService(t)
	ctx := context.Background()
	userID := "user-123"
	lat, lon := 55.75, 37.61
	foundIncidents := []*models.Incident{
		{ID: uuid.New(), Name: "Зона А"},
	}

	// Ожидания
	// 1. Поиск активной локации
	repoMock.EXPECT().
		FindActiveLocation(ctx, lat, lon).
		Return(foundIncidents, nil).
		Times(1)

	// 2. Сохранение факта проверки
	repoMock.EXPECT().
		SaveLocationCheck(ctx, gomock.Any()).
		// Проверяем, что сохраняем "опасную" проверку
		Do(func(ctx context.Context, check *models.LocationCheck) {
			assert.True(t, check.IsDangerous)
			assert.Equal(t, userID, check.UserID)
		}).Return(nil).Times(1)

	// 3. Публикация вебхука
	webhookMock.EXPECT().
		Publish(ctx, gomock.Any()).
		// Проверяем, что событие вебхука опасное и содержит инциденты
		Do(func(ctx context.Context, event webhook.WebhookEvent) {
			assert.True(t, event.IsDangerous)
			assert.Equal(t, userID, event.UserID)
			assert.Equal(t, foundIncidents, event.Incidents)
		}).Return(nil).Times(1)

	// Действие
	incidents, err := service.CheckLocation(ctx, userID, lat, lon)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, foundIncidents, incidents)
}

func TestCheckLocation_Safe(t *testing.T) {
	// Подготовка
	service, repoMock, webhookMock := newTestIncidentService(t)
	ctx := context.Background()
	userID := "user-456"
	lat, lon := 50.0, 50.0
	var foundIncidents []*models.Incident // Пустой слайс

	// Ожидания
	// 1. Поиск активной локации ничего не возвращает
	repoMock.EXPECT().
		FindActiveLocation(ctx, lat, lon).
		Return(foundIncidents, nil).
		Times(1)

	// 2. Сохранение факта проверки
	repoMock.EXPECT().
		SaveLocationCheck(ctx, gomock.Any()).
		Do(func(ctx context.Context, check *models.LocationCheck) {
			assert.False(t, check.IsDangerous)
			assert.Equal(t, userID, check.UserID)
		}).Return(nil).Times(1)

	// 3. Публикатор вебхуков НЕ вызывается
	webhookMock.EXPECT().Publish(gomock.Any(), gomock.Any()).Times(0)

	// Действие
	incidents, err := service.CheckLocation(ctx, userID, lat, lon)

	// Проверки
	require.NoError(t, err)
	assert.Empty(t, incidents)
}

func TestGetStats_Success(t *testing.T) {
	// Подготовка
	service, repoMock, _ := newTestIncidentService(t)
	ctx := context.Background()
	expectedUserCount := 42

	// Ожидания
	repoMock.EXPECT().GetLocationCheckStats(ctx, service.cfg.StatsTimeWindowMinutes).Return(expectedUserCount, nil).Times(1)

	// Действие
	count, err := service.GetStats(ctx)

	// Проверки
	require.NoError(t, err)
	assert.Equal(t, expectedUserCount, count)
}
