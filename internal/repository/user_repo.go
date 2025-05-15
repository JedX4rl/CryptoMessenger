package repository

import (
	"CryptoMessenger/internal/domain"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
)

type UserRepository struct {
	db *sql.DB
}

func (u *UserRepository) Create(ctx context.Context, user domain.User) error {
	query := "INSERT INTO users (user_id, username, password_hash) VAlUES ($1, $2, $3)"
	_, err := u.db.ExecContext(ctx, query, user.ID, user.Username, user.PasswordHash)
	if err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("error while inserting user: %w", err)
	}
	return nil
}

func (u *UserRepository) GetByUsername(ctx context.Context, username string) (domain.User, error) {
	query := "SELECT user_id, username, password_hash FROM users WHERE username = $1"

	var user domain.User

	row := u.db.QueryRowContext(ctx, query, username)
	if err := row.Err(); err != nil {
		slog.Error("error while getting user by username", err, username)
		slog.Info("username", username)
		return domain.User{}, fmt.Errorf("error getting user by username: %w", err)
	}
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {

		slog.Error("error while getting user by username", err)
		return domain.User{}, fmt.Errorf("error getting user by username: %w", err)
	}
	return user, nil
}

func (u *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	query := "SELECT user_id, username, password_hash FROM users WHERE user_id = $1"

	var user domain.User

	row := u.db.QueryRowContext(ctx, query, id)
	if err := row.Err(); err != nil {
		return domain.User{}, fmt.Errorf("error getting user by id: %w", err)
	}
	if err := row.Scan(&user.ID, &user.Username, &user.PasswordHash); err != nil {
		return domain.User{}, fmt.Errorf("error getting user by id: %w", err)
	}
	return user, nil
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}
