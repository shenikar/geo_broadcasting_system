package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shenikar/geo_broadcasting_system/internal/config"
	"github.com/shenikar/geo_broadcasting_system/internal/models"
	"github.com/shenikar/geo_broadcasting_system/internal/service/mocks"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newTestHandler создает новый экземпляр Handler с мокированным сервисом
func newTestHandler(t *testing.T) (*Handler, *mocks.MockIncidentService, *gin.Engine) {
	ctrl := gomock.NewController(t)
	mockService := mocks.NewMockIncidentService(ctrl)

	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{}) // Отключаем вывод логов в тестах

	cfg := &config.Config{
		APIKeys:                []string{"test-api-key"},
		StatsTimeWindowMinutes: 60,
	}

	handler := NewHandler(mockService, logger, cfg)

	// Настройка Gin роутера для тестов
	gin.SetMode(gin.TestMode)
	router := gin.New()
	api := router.Group("/api/v1")
	handler.RegisterRoutes(api)

	return handler, mockService, router
}

// makeRequest - вспомогательная функция для выполнения HTTP-запросов
func makeRequest(router *gin.Engine, method, url string, body io.Reader, headers ...map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, body)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, h := range headers {
		for key, value := range h {
			req.Header.Set(key, value)
		}
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCreateIncident_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	reqBody := CreateIncidentRequest{
		Name:         "Test Incident",
		Description:  "Description",
		Latitude:     10.0,
		Longitude:    20.0,
		RadiusMeters: 100,
	}
	expectedIncident := &models.Incident{
		ID:           incidentID,
		Name:         reqBody.Name,
		Description:  reqBody.Description,
		Latitude:     reqBody.Latitude,
		Longitude:    reqBody.Longitude,
		RadiusMeters: reqBody.RadiusMeters,
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	mockService.EXPECT().
		CreateIncident(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, inc *models.Incident) error {
			*inc = *expectedIncident // Обновляем переданный инцидент
			return nil
		}).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/incidents", bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp IncidentResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, incidentID, resp.ID)
	assert.Equal(t, reqBody.Name, resp.Name)
}

func TestCreateIncident_InvalidJSON(t *testing.T) {
	_, mockService, router := newTestHandler(t)

	mockService.EXPECT().CreateIncident(gomock.Any(), gomock.Any()).Times(0) // Сервис не должен вызываться

	w := makeRequest(router, "POST", "/api/v1/incidents", bytes.NewBufferString(`{"name": "test"`), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestCreateIncident_ValidationError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := CreateIncidentRequest{ // Отсутствует Name
		Description:  "Description",
		Latitude:     10.0,
		Longitude:    20.0,
		RadiusMeters: 100,
	}

	mockService.EXPECT().CreateIncident(gomock.Any(), gomock.Any()).Times(0) // Сервис не должен вызываться

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/incidents", bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Error:Field validation for 'Name' failed on the 'required' tag")
}

func TestCreateIncident_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := CreateIncidentRequest{
		Name:         "Test Incident",
		Description:  "Description",
		Latitude:     10.0,
		Longitude:    20.0,
		RadiusMeters: 100,
	}
	serviceError := errors.New("failed to create incident in service")

	mockService.EXPECT().
		CreateIncident(gomock.Any(), gomock.Any()).
		Return(serviceError).
		Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/incidents", bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestGetIncident_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	expectedIncident := &models.Incident{
		ID:           incidentID,
		Name:         "Retrieved Incident",
		Latitude:     30.0,
		Longitude:    40.0,
		RadiusMeters: 200,
		Status:       "active",
	}

	mockService.EXPECT().GetIncident(gomock.Any(), incidentID).Return(expectedIncident, nil).Times(1)

	w := makeRequest(router, "GET", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp IncidentResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, incidentID, resp.ID)
	assert.Equal(t, expectedIncident.Name, resp.Name)
}

func TestGetIncident_InvalidID(t *testing.T) {
	_, mockService, router := newTestHandler(t)

	mockService.EXPECT().GetIncident(gomock.Any(), gomock.Any()).Times(0) // Сервис не должен вызываться

	w := makeRequest(router, "GET", "/api/v1/incidents/invalid-uuid", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid incident ID")
}

func TestGetIncident_NotFound(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	serviceError := errors.New("incident not found")

	mockService.EXPECT().GetIncident(gomock.Any(), incidentID).Return(nil, serviceError).Times(1)

	w := makeRequest(router, "GET", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "incident not found")
}

func TestGetIncident_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	serviceError := errors.New("database error")

	mockService.EXPECT().GetIncident(gomock.Any(), incidentID).Return(nil, serviceError).Times(1)

	w := makeRequest(router, "GET", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusNotFound, w.Code) // Хендлер возвращает 404 для всех ошибок сервиса при получении инцидента
	assert.Contains(t, w.Body.String(), "incident not found")
}

func TestListIncidents_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	expectedIncidents := []*models.Incident{
		{ID: uuid.New(), Name: "Incident 1", Status: "active"},
		{ID: uuid.New(), Name: "Incident 2", Status: "inactive"},
	}

	mockService.EXPECT().ListIncidents(gomock.Any(), 1, 10).Return(expectedIncidents, nil).Times(1)

	w := makeRequest(router, "GET", "/api/v1/incidents?page=1&pageSize=10", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []IncidentResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, expectedIncidents[0].Name, resp[0].Name)
}

func TestListIncidents_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	serviceError := errors.New("failed to list incidents")

	mockService.EXPECT().ListIncidents(gomock.Any(), 1, 10).Return(nil, serviceError).Times(1)

	w := makeRequest(router, "GET", "/api/v1/incidents?page=1&pageSize=10", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestUpdateIncident_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	reqBody := UpdateIncidentRequest{
		Name:         "Updated Name",
		Description:  "Updated Description",
		Latitude:     11.0,
		Longitude:    21.0,
		RadiusMeters: 110,
		Status:       "active",
	}

	mockService.EXPECT().
		UpdateIncident(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, inc *models.Incident) error {
			assert.Equal(t, incidentID, inc.ID)
			assert.Equal(t, reqBody.Name, inc.Name)
			return nil
		}).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "PUT", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUpdateIncident_InvalidID(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := UpdateIncidentRequest{
		Name:         "Updated Name",
		Latitude:     11.0,
		Longitude:    21.0,
		RadiusMeters: 110,
		Status:       "active",
	}

	mockService.EXPECT().UpdateIncident(gomock.Any(), gomock.Any()).Times(0)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "PUT", "/api/v1/incidents/invalid-uuid", bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid incident ID")
}

func TestUpdateIncident_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	reqBody := UpdateIncidentRequest{
		Name:         "Updated Name",
		Description:  "Updated Description",
		Latitude:     11.0,
		Longitude:    21.0,
		RadiusMeters: 110,
		Status:       "active",
	}
	serviceError := errors.New("failed to update incident")

	mockService.EXPECT().UpdateIncident(gomock.Any(), gomock.Any()).Return(serviceError).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "PUT", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), bytes.NewBuffer(bodyBytes), map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusInternalServerError, w.Code) // Ожидаем 500, так как валидация пройдена
	assert.Contains(t, w.Body.String(), "failed to update incident in service")
}

func TestDeleteIncident_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()

	mockService.EXPECT().DeactivateIncident(gomock.Any(), incidentID).Return(nil).Times(1)

	w := makeRequest(router, "DELETE", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteIncident_InvalidID(t *testing.T) {
	_, mockService, router := newTestHandler(t)

	mockService.EXPECT().DeactivateIncident(gomock.Any(), gomock.Any()).Times(0)

	w := makeRequest(router, "DELETE", "/api/v1/incidents/invalid-uuid", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid incident ID")
}

func TestDeleteIncident_NotFound(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	incidentID := uuid.New()
	serviceError := errors.New("incident not found for deactivate")

	mockService.EXPECT().DeactivateIncident(gomock.Any(), incidentID).Return(serviceError).Times(1)

	w := makeRequest(router, "DELETE", fmt.Sprintf("/api/v1/incidents/%s", incidentID.String()), nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusInternalServerError, w.Code) // Хендлер возвращает 500 для этой ошибки
	assert.Contains(t, w.Body.String(), "failed to deactivate incident")
}

func TestCheckLocation_Success_Danger(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := LocationCheckRequest{
		UserID:    "user123",
		Latitude:  50.0,
		Longitude: 50.0,
	}
	incidentsFound := []*models.Incident{
		{ID: uuid.New(), Name: "Danger Zone A"},
	}

	mockService.EXPECT().CheckLocation(gomock.Any(), reqBody.UserID, reqBody.Latitude, reqBody.Longitude).Return(incidentsFound, nil).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/location/check", bytes.NewBuffer(bodyBytes))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []IncidentResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, incidentsFound[0].Name, resp[0].Name)
}

func TestCheckLocation_Success_Safe(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := LocationCheckRequest{
		UserID:    "user123",
		Latitude:  50.0,
		Longitude: 50.0,
	}
	var incidentsFound []*models.Incident // No incidents found

	mockService.EXPECT().CheckLocation(gomock.Any(), reqBody.UserID, reqBody.Latitude, reqBody.Longitude).Return(incidentsFound, nil).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/location/check", bytes.NewBuffer(bodyBytes))

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []IncidentResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Empty(t, resp)
}

func TestCheckLocation_ValidationError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := LocationCheckRequest{ // Отсутствует UserID
		Latitude:  50.0,
		Longitude: 50.0,
	}

	mockService.EXPECT().CheckLocation(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/location/check", bytes.NewBuffer(bodyBytes))

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Error:Field validation for 'UserID' failed on the 'required' tag")
}

func TestCheckLocation_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	reqBody := LocationCheckRequest{
		UserID:    "user123",
		Latitude:  50.0,
		Longitude: 50.0,
	}
	serviceError := errors.New("failed to check location")

	mockService.EXPECT().CheckLocation(gomock.Any(), reqBody.UserID, reqBody.Latitude, reqBody.Longitude).Return(nil, serviceError).Times(1)

	bodyBytes, _ := json.Marshal(reqBody)
	w := makeRequest(router, "POST", "/api/v1/location/check", bytes.NewBuffer(bodyBytes))

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestGetStats_Success(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	expectedCount := 123

	mockService.EXPECT().GetStats(gomock.Any()).Return(expectedCount, nil).Times(1)

	w := makeRequest(router, "GET", "/api/v1/incidents/stats", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusOK, w.Code)
	var resp StatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, expectedCount, resp.UserCount)
}

func TestGetStats_ServiceError(t *testing.T) {
	_, mockService, router := newTestHandler(t)
	serviceError := errors.New("failed to get stats")

	mockService.EXPECT().GetStats(gomock.Any()).Return(0, serviceError).Times(1)

	w := makeRequest(router, "GET", "/api/v1/incidents/stats", nil, map[string]string{"X-API-Key": "test-api-key"})

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "internal server error")
}

func TestHealthCheck_Success(t *testing.T) {
	_, _, router := newTestHandler(t)

	w := makeRequest(router, "GET", "/api/v1/system/health", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"ok"`)
}

func TestAPIKeyAuthMiddleware_Success(t *testing.T) {
	// Создаем Gin-роутер и добавляем middleware
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{})

	cfg := &config.Config{
		APIKeys: []string{"valid-key"},
	}

	router.Use(APIKeyAuthMiddleware(cfg, logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := makeRequest(router, "GET", "/test", nil, map[string]string{"X-API-Key": "valid-key"})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAPIKeyAuthMiddleware_MissingKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{})

	cfg := &config.Config{
		APIKeys: []string{"valid-key"},
	}

	router.Use(APIKeyAuthMiddleware(cfg, logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := makeRequest(router, "GET", "/test", nil) // Нет API ключа
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "API key required")
}

func TestAPIKeyAuthMiddleware_InvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	logger := logrus.New()
	logger.SetOutput(&bytes.Buffer{})

	cfg := &config.Config{
		APIKeys: []string{"valid-key"},
	}

	router.Use(APIKeyAuthMiddleware(cfg, logger))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	w := makeRequest(router, "GET", "/test", nil, map[string]string{"X-API-Key": "invalid-key"})
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key")
}
