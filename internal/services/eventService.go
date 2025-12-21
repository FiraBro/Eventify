package services

import (
	"local-go/internal/models"
	"local-go/internal/repositories"
)

type EventService struct {
	repo *repositories.EventRepository
}

func NewEventService(repo *repositories.EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) CreateEvent(event *models.Event) error {
	return s.repo.Create(event)
}

func (s *EventService) GetAllEvents() ([]models.Event, error) {
	return s.repo.GetAll()
}

func (s *EventService) GetEventByID(id string) (*models.Event, error) {
	return s.repo.GetByID(id)
}

func (s *EventService) UpdateEvent(event *models.Event) error {
	return s.repo.Update(event)
}

func (s *EventService) DeleteEvent(id string) error {
	return s.repo.Delete(id)
}
