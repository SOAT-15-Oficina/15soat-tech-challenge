package service

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

type VehicleService interface {
	Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error)
	GetAll(ctx context.Context) ([]domain.Vehicle, error)
	Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type vehicleService struct {
	repo repository.VehicleRepository
}

func NewVehicleService(repo repository.VehicleRepository) VehicleService {
	return &vehicleService{repo: repo}
}

func (s *vehicleService) Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	return s.repo.Create(ctx, vehicle)
}

func (s *vehicleService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *vehicleService) GetAll(ctx context.Context) ([]domain.Vehicle, error) {
	return s.repo.FindAll(ctx)
}

func (s *vehicleService) Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	return s.repo.Update(ctx, vehicle)
}

func (s *vehicleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
