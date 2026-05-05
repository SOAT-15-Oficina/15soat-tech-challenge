package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockUserRepo mocks repository.UserRepository
type mockUserRepo struct {
	mock.Mock
}

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- hashPassword / verifyPassword ---

func TestHashAndVerifyPassword_RoundTrip(t *testing.T) {
	hash, err := hashPassword("my-secret")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	err = verifyPassword("my-secret", hash)
	assert.NoError(t, err)
}

func TestVerifyPassword_WrongPassword(t *testing.T) {
	hash, err := hashPassword("correct")
	require.NoError(t, err)

	err = verifyPassword("wrong", hash)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestVerifyPassword_InvalidHashFormat(t *testing.T) {
	err := verifyPassword("any", "not-a-valid-hash")
	assert.Error(t, err)
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	created := &domain.User{ID: uuid.New(), Username: "alice", Role: domain.UserRoleAdmin}

	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(created, nil)

	result, err := svc.Register(ctx, "alice", "pass123", domain.UserRoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, "alice", result.Username)
	repo.AssertExpectations(t)
}

func TestRegister_EmptyUsername(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")

	result, err := svc.Register(context.Background(), "", "pass", domain.UserRoleAdmin)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestRegister_EmptyPassword(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")

	result, err := svc.Register(context.Background(), "alice", "", domain.UserRoleAdmin)
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestRegister_InvalidRole(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")

	result, err := svc.Register(context.Background(), "alice", "pass", "superuser")
	assert.Error(t, err)
	assert.Nil(t, result)
	repo.AssertNotCalled(t, "Create")
}

func TestRegister_EmployeeRole(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	created := &domain.User{ID: uuid.New(), Username: "bob", Role: domain.UserRoleEmployee}

	repo.On("Create", ctx, mock.AnythingOfType("*domain.User")).Return(created, nil)

	result, err := svc.Register(ctx, "bob", "pass", domain.UserRoleEmployee)
	require.NoError(t, err)
	assert.Equal(t, domain.UserRoleEmployee, result.Role)
}

// --- Login ---

func TestLogin_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "jwt-secret")
	ctx := context.Background()

	hash, _ := hashPassword("correct-pass")
	user := &domain.User{ID: uuid.New(), Username: "alice", PasswordHash: hash, Role: domain.UserRoleAdmin}

	repo.On("FindByUsername", ctx, "alice").Return(user, nil)

	token, err := svc.Login(ctx, "alice", "correct-pass")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestLogin_UserNotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "jwt-secret")
	ctx := context.Background()

	repo.On("FindByUsername", ctx, "ghost").Return(nil, pgx.ErrNoRows)

	token, err := svc.Login(ctx, "ghost", "any")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "jwt-secret")
	ctx := context.Background()

	hash, _ := hashPassword("correct")
	user := &domain.User{ID: uuid.New(), Username: "alice", PasswordHash: hash, Role: domain.UserRoleAdmin}

	repo.On("FindByUsername", ctx, "alice").Return(user, nil)

	token, err := svc.Login(ctx, "alice", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

func TestLogin_RepoError(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "jwt-secret")
	ctx := context.Background()

	repo.On("FindByUsername", ctx, "alice").Return(nil, errors.New("db down"))

	token, err := svc.Login(ctx, "alice", "pass")
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrInvalidCredentials)
	assert.Empty(t, token)
}

// --- Update ---

func TestUserUpdate_InvalidRole(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()
	existing := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}

	repo.On("FindByID", ctx, id).Return(existing, nil)

	result, err := svc.Update(ctx, id, "alice", "hacker")
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserUpdate_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()
	existing := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}
	updated := &domain.User{ID: id, Username: "alice2", Role: domain.UserRoleEmployee}

	repo.On("FindByID", ctx, id).Return(existing, nil)
	repo.On("Update", ctx, mock.AnythingOfType("*domain.User")).Return(updated, nil)

	result, err := svc.Update(ctx, id, "alice2", domain.UserRoleEmployee)
	require.NoError(t, err)
	assert.Equal(t, "alice2", result.Username)
}

func TestUserGetByID_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()
	user := &domain.User{ID: id, Username: "alice", Role: domain.UserRoleAdmin}

	repo.On("FindByID", ctx, id).Return(user, nil)

	result, err := svc.GetByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, result.ID)
}

func TestUserGetByID_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()

	repo.On("FindByID", ctx, id).Return(nil, pgx.ErrNoRows)

	result, err := svc.GetByID(ctx, id)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestUserGetAll_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()

	repo.On("FindAll", ctx).Return([]domain.User{{ID: uuid.New(), Username: "alice"}}, nil)

	results, err := svc.GetAll(ctx)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
}

func TestUserDelete_Success(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(nil)

	err := svc.Delete(ctx, id)
	assert.NoError(t, err)
}

func TestUserDelete_Error(t *testing.T) {
	repo := new(mockUserRepo)
	svc := NewUserService(repo, "secret")
	ctx := context.Background()
	id := uuid.New()

	repo.On("Delete", ctx, id).Return(errors.New("db error"))

	err := svc.Delete(ctx, id)
	assert.Error(t, err)
}
