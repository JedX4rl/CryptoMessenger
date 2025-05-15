package service

import (
	"CryptoMessenger/internal/domain"
	myErrors "CryptoMessenger/internal/errors"
	natsjs "CryptoMessenger/internal/infrastructure/nats"
	"CryptoMessenger/internal/repository"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users    repository.UserRepo
	jsClient *natsjs.JSClient
}

func NewAuthService(userRepo repository.UserRepo, jsClient *natsjs.JSClient) *AuthService {
	return &AuthService{
		users:    userRepo,
		jsClient: jsClient,
	}
}

func (s *AuthService) Register(ctx context.Context, username, password string) (string, error) {
	if _, err := s.users.GetByUsername(ctx, username); err == nil {
		return "", myErrors.ErrUserExists
	} else if !errors.Is(err, sql.ErrNoRows) {
		fmt.Println(err.Error())
		return "", fmt.Errorf("error getting user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hashing failed: %w", err)
	}

	uid := uuid.New().String()
	user := domain.User{
		ID:           uid,
		Username:     username,
		PasswordHash: string(hash),
	}

	if err = s.users.Create(ctx, user); err != nil {
		return "", fmt.Errorf("error creating user: %w", err)
	}

	if err = s.jsClient.EnsureInvitesConsumer(uid); err != nil {
		return "", fmt.Errorf("failed to init invites consumer: %w", err)
	}

	if err = s.jsClient.EnsureInviteReactionsConsumer(uid); err != nil {
		return "", fmt.Errorf("failed to init invite reactions consumer: %w", err)
	}

	return uid, nil
}

func (s *AuthService) Login(ctx context.Context, username, password string) (string, error) {
	user, err := s.users.GetByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("invalid username: %w", err)
	}
	if err != nil {
		return "", fmt.Errorf("some error occured: %w", err)
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", myErrors.ErrInvalidPassword
	}
	return user.ID, nil
}
