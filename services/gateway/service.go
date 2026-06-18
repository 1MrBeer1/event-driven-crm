package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/mr-beer/event-driven-crm/internal/auth"
	"github.com/mr-beer/event-driven-crm/internal/models"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrWeakPassword       = errors.New("password must contain at least 8 characters")
)

type AuthService struct {
	users  UserRepository
	tokens *auth.Manager
}

func NewAuthService(users UserRepository, tokens *auth.Manager) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (string, models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || !strings.Contains(email, "@") {
		return "", models.User{}, fmt.Errorf("invalid email")
	}
	if len(password) < 8 {
		return "", models.User{}, ErrWeakPassword
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return "", models.User{}, err
	}

	user := models.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		Role:         "manager",
	}

	user, err = s.users.Create(ctx, user)
	if err != nil {
		return "", models.User{}, err
	}

	token, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	return token, user, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, models.User, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return "", models.User{}, ErrInvalidCredentials
	}

	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return "", models.User{}, ErrInvalidCredentials
	}

	token, err := s.tokens.Generate(user.ID, user.Email, user.Role)
	return token, user, err
}
