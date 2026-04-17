package service

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

type SupplyService interface {
	Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error)
	GetAll(ctx context.Context) ([]domain.Supply, error)
	Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type supplyService struct {
	repo repository.SupplyRepository
}

func NewSupplyService(repo repository.SupplyRepository) SupplyService {
	return &supplyService{repo: repo}
}

func (s *supplyService) Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	return s.repo.Create(ctx, supply)
}

func (s *supplyService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *supplyService) GetAll(ctx context.Context) ([]domain.Supply, error) {
	return s.repo.FindAll(ctx)
}

func (s *supplyService) Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	return s.repo.Update(ctx, supply)
}

func (s *supplyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
