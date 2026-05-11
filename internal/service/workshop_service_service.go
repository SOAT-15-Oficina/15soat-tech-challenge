package service

import (
	"context"
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

var ErrWorkshopServiceTitleAlreadyExists = errors.New("service title already exists")

type DeleteWorkshopServiceResult struct {
	Deleted             bool
	Deactivated         bool
	DeactivatedResource *domain.WorkshopService
}

type WorkshopServiceUpdateInput struct {
	Title                *string
	Description          *string
	PriceCents           *int
	EstimatedTimeMinutes *int
	Active               *bool
}

type WorkshopServiceService interface {
	Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error)
	List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error)
	Update(ctx context.Context, id uuid.UUID, input WorkshopServiceUpdateInput) (*domain.WorkshopService, error)
	Delete(ctx context.Context, id uuid.UUID) (*DeleteWorkshopServiceResult, error)
}

type workshopServiceService struct {
	repo repository.WorkshopServiceRepository
}

func NewWorkshopServiceService(repo repository.WorkshopServiceRepository) WorkshopServiceService {
	return &workshopServiceService{repo: repo}
}

func (s *workshopServiceService) Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	ws.Active = true

	if err := ws.Validate(); err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByTitle(ctx, ws.Title, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWorkshopServiceTitleAlreadyExists
	}

	return s.repo.Create(ctx, ws)
}

func (s *workshopServiceService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *workshopServiceService) List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error) {
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 10
	}

	return s.repo.List(ctx, filters)
}

func (s *workshopServiceService) Update(ctx context.Context, id uuid.UUID, input WorkshopServiceUpdateInput) (*domain.WorkshopService, error) {
	current, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.Title != nil {
		current.Title = *input.Title
	}
	if input.Description != nil {
		current.Description = *input.Description
	}
	if input.PriceCents != nil {
		current.PriceCents = *input.PriceCents
	}
	if input.EstimatedTimeMinutes != nil {
		current.EstimatedTimeMinutes = *input.EstimatedTimeMinutes
	}
	if input.Active != nil {
		current.Active = *input.Active
	}

	if err := current.Validate(); err != nil {
		return nil, err
	}

	exists, err := s.repo.ExistsByTitle(ctx, current.Title, &id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrWorkshopServiceTitleAlreadyExists
	}

	updated, err := s.repo.Update(ctx, current)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *workshopServiceService) Delete(ctx context.Context, id uuid.UUID) (*DeleteWorkshopServiceResult, error) {
	_, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	hasLinks, err := s.repo.HasWorkOrderLinks(ctx, id)
	if err != nil {
		return nil, err
	}

	if hasLinks {
		deactivated, err := s.repo.Deactivate(ctx, id)
		if err != nil {
			return nil, err
		}

		return &DeleteWorkshopServiceResult{
			Deactivated:         true,
			DeactivatedResource: deactivated,
		}, nil
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, err
	}

	return &DeleteWorkshopServiceResult{Deleted: true}, nil
}

