package v1

import "github.com/shenikar/geo_broadcasting_system/internal/models"

// DTOToIncidentModel преобразует DTO создания/обновления в доменную модель.
// Используем одну функцию, так как поля совпадают.
func DTOToIncidentModel(dto any) *models.Incident {
	switch v := dto.(type) {
	case CreateIncidentRequest:
		return &models.Incident{
			Name:         v.Name,
			Description:  v.Description,
			Latitude:     v.Latitude,
			Longitude:    v.Longitude,
			RadiusMeters: v.RadiusMeters,
		}
	case UpdateIncidentRequest:
		return &models.Incident{
			Name:         v.Name,
			Description:  v.Description,
			Latitude:     v.Latitude,
			Longitude:    v.Longitude,
			RadiusMeters: v.RadiusMeters,
			Status:       v.Status,
		}
	}
	return nil
}

// ModelToIncidentResponse преобразует доменную модель в DTO для ответа
func ModelToIncidentResponse(model *models.Incident) *IncidentResponse {
	return &IncidentResponse{
		ID:           model.ID,
		Name:         model.Name,
		Description:  model.Description,
		Latitude:     model.Latitude,
		Longitude:    model.Longitude,
		RadiusMeters: model.RadiusMeters,
		Status:       model.Status,
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

// ModelsToIncidentResponses преобразует слайс моделей в слайс DTO
func ModelsToIncidentResponses(models []*models.Incident) []*IncidentResponse {
	responses := make([]*IncidentResponse, len(models))
	for i, model := range models {
		responses[i] = ModelToIncidentResponse(model)
	}
	return responses
}
