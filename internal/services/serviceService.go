package services

import (
	"github.com/FiraBro/local-go/internal/models"
	"github.com/FiraBro/local-go/internal/repositories"
)

type ServiceService struct {
	repo *repositories.ServiceRepository
}

func NewServiceService(repo *repositories.ServiceRepository) *ServiceService {
	return &ServiceService{repo: repo}
}

func (s *ServiceService) Create(service *models.Service) error {
	return s.repo.Create(service)
}

func (s *ServiceService) GetAll() ([]models.Service, error) {
	return s.repo.GetAll()
}

func (s *ServiceService) GetByID(id string) (*models.Service, error) {
	return s.repo.GetByID(id)
}

func (s *ServiceService) Update(id string, service *models.Service) error {
	return s.repo.Update(id, service)
}

func (s *ServiceService) Delete(id string) error {
	return s.repo.Delete(id)
}

func (s *ServiceService) GetCategories() ([]string, error) {
	return s.repo.GetCategories()
}
