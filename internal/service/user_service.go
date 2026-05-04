package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/auth"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/argon2"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

const (
	argon2Time    = 1
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

func hashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argon2Memory, argon2Time, argon2Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

func verifyPassword(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return fmt.Errorf("invalid hash format: unexpected number of parts")
	}

	if parts[1] != "argon2id" {
		return fmt.Errorf("invalid hash format: unsupported algorithm")
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return fmt.Errorf("invalid hash format: %w", err)
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return fmt.Errorf("invalid hash format: %w", err)
	}

	saltB64 := parts[4]
	hashB64 := parts[5]

	salt, err := base64.RawStdEncoding.DecodeString(saltB64)
	if err != nil {
		return fmt.Errorf("invalid salt encoding: %w", err)
	}
	expectedHash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return fmt.Errorf("invalid hash encoding: %w", err)
	}

	computed := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))
	if subtle.ConstantTimeCompare(computed, expectedHash) != 1 {
		return ErrInvalidCredentials
	}
	return nil
}

type UserService interface {
	Register(ctx context.Context, username, password string, role domain.UserRole) (*domain.User, error)
	Login(ctx context.Context, username, password string) (string, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetAll(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type userService struct {
	repo         repository.UserRepository
	jwtSecretKey string
}

func NewUserService(repo repository.UserRepository, jwtSecretKey string) UserService {
	return &userService{repo: repo, jwtSecretKey: jwtSecretKey}
}

func (s *userService) Register(ctx context.Context, username, password string, role domain.UserRole) (*domain.User, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required")
	}
	if role != domain.UserRoleAdmin && role != domain.UserRoleEmployee {
		return nil, fmt.Errorf("invalid role: must be 'admin' or 'employee'")
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &domain.User{
		Username:     username,
		PasswordHash: string(hash),
		Role:         role,
	}
	return s.repo.Create(ctx, user)
}

func (s *userService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrInvalidCredentials
		}
		return "", err
	}

	if err := verifyPassword(password, user.PasswordHash); err != nil {
		return "", ErrInvalidCredentials
	}

	return auth.GenerateToken(user.Username, string(user.Role), s.jwtSecretKey)
}

func (s *userService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *userService) GetAll(ctx context.Context) ([]domain.User, error) {
	return s.repo.FindAll(ctx)
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error) {
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if username != "" {
		existing.Username = username
	}
	if role != "" {
		if role != domain.UserRoleAdmin && role != domain.UserRoleEmployee {
			return nil, fmt.Errorf("invalid role: must be 'admin' or 'employee'")
		}
		existing.Role = role
	}

	return s.repo.Update(ctx, existing)
}

func (s *userService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
