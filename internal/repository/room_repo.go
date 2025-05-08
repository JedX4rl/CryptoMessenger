package repository

import (
	"CryptoMessenger/internal/domain"
	"context"
	"database/sql"
	"fmt"
)

type RoomRepository struct {
	db *sql.DB
}

func (r *RoomRepository) Create(ctx context.Context, cfg domain.RoomConfig) error {
	query := "INSERT INTO chats (chat_id, name, algorithm, mode, padding, iv, random_delta) VALUES ($1, $2, $3, $4, $5, $6, $7)"
	_, err := r.db.ExecContext(ctx, query, cfg.RoomID, cfg.RoomName, cfg.Algorithm, cfg.Mode, cfg.Padding, cfg.Iv, cfg.RandomDelta)
	if err != nil {
		return fmt.Errorf("error creating room: %w", err)
	}
	return nil
}

func (r *RoomRepository) Delete(ctx context.Context, roomID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *RoomRepository) Get(ctx context.Context, roomID string) (domain.RoomConfig, error) {
	//TODO implement me
	panic("implement me")
}

func (r *RoomRepository) AddMember(ctx context.Context, roomID, userID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *RoomRepository) RemoveMember(ctx context.Context, roomID, userID string) error {
	//TODO implement me
	panic("implement me")
}

func (r *RoomRepository) ListMembers(ctx context.Context, roomID string) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func NewRoomRepository(db *sql.DB) *RoomRepository {
	return &RoomRepository{
		db: db,
	}
}
