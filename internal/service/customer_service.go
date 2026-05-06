package service

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

type CustomerService interface {
	Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
	GetAll(ctx context.Context) ([]domain.Customer, error)
	GetAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error)
	Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type customerService struct {
	repo repository.CustomerRepository
}

func NewCustomerService(repo repository.CustomerRepository) CustomerService {
	return &customerService{repo: repo}
}

func (s *customerService) Create(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	customer.Normalize()
	if err := customer.ValidateDocument(); err != nil {
		return nil, err
	}
	return s.repo.Create(ctx, customer)
}

func (s *customerService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *customerService) GetAll(ctx context.Context) ([]domain.Customer, error) {
	return s.repo.FindAll(ctx)
}

func (s *customerService) GetAllWithFilters(ctx context.Context, filters domain.CustomerListFilters) ([]domain.Customer, error) {
	return s.repo.FindAllWithFilters(ctx, filters)
}

func (s *customerService) Update(ctx context.Context, customer *domain.Customer) (*domain.Customer, error) {
	customer.Normalize()
	if err := customer.ValidateDocument(); err != nil {
		return nil, err
	}
	return s.repo.Update(ctx, customer)
}

func (s *customerService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
