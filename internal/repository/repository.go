package repository

import (
	"CryptoMessenger/internal/domain"
	"context"
	"database/sql"
)

type KeyRepo interface {
	Store(ctx context.Context, pk domain.PublicKey) error
	ListByRoom(ctx context.Context, roomID string) ([]domain.PublicKey, error)
}

type RoomRepo interface {
	Create(ctx context.Context, r domain.RoomConfig) error
	Delete(ctx context.Context, roomID string) error
	Get(ctx context.Context, roomID string) (domain.RoomConfig, error)

	AddMember(ctx context.Context, roomID, userID string) error
	RemoveMember(ctx context.Context, roomID, userID string) error
	ListMembers(ctx context.Context, roomID string) ([]string, error)
}

type UserRepo interface {
	Create(ctx context.Context, u domain.User) error
	GetByUsername(ctx context.Context, username string) (domain.User, error)
	GetByID(ctx context.Context, id string) (domain.User, error)
}

type Repository struct {
	KeyRepo
	RoomRepo
	UserRepo
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		KeyRepo:  NewKeyRepository(db),
		RoomRepo: NewRoomRepository(db),
		UserRepo: NewUserRepository(db),
	}

}
