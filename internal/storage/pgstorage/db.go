package pgstorage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

type PgStorage struct {
	db *pgxpool.Pool
}

func NewStorage(dsn string) (*PgStorage, error) {
	ctx := context.Background()
	dbConfig, dbErr := pgxpool.ParseConfig(dsn)

	if dbErr != nil {
		return nil, dbErr
	}

	db, err := pgxpool.NewWithConfig(ctx, dbConfig)

	if err != nil {
		return nil, err
	}

	if err := db.Ping(ctx); err != nil {
		return nil, err
	}

	err = checkTables(ctx, db)
	if err != nil {
		return nil, err
	}

	return &PgStorage{db}, nil
}

func checkTables(ctx context.Context, db *pgxpool.Pool) error {
	sql, err := os.ReadFile("internal/storage/pgstorage/db.sql")
	if err != nil {
		return err
	}
	code := string(sql)
	_, err = db.Exec(ctx, code)
	if err != nil {
		return fmt.Errorf("create tables error: %w", err)
	}
	return nil
}
