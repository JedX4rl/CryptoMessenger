package repository

import (
	"CryptoMessenger/internal/domain"
	"context"
	"database/sql"
)

type KeyRepository struct {
	db *sql.DB
}

func (k *KeyRepository) Store(ctx context.Context, pk domain.PublicKey) error {
	//TODO implement me
	panic("implement me")
}

func (k *KeyRepository) ListByRoom(ctx context.Context, roomID string) ([]domain.PublicKey, error) {
	//TODO implement me
	panic("implement me")
}

func NewKeyRepository(db *sql.DB) *KeyRepository {
	return &KeyRepository{
		db: db,
	}
}
