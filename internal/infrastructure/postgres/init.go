package postgres

import (
	"CryptoMessenger/internal/config/storageConfig"
	"database/sql"
	"fmt"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	"time"
)

func NewStorage(dbConfig *storageConfig.Config) (*sql.DB, error) {

	connectionString := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=%s",
		dbConfig.Username, dbConfig.Password,
		dbConfig.Host, dbConfig.Port,
		dbConfig.DBName, dbConfig.SSLMode,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(12)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(20 * time.Minute)
	db.SetConnMaxLifetime(10 * time.Minute)

	return db, nil
}
