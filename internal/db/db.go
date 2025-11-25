package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB holds the database connection pool
var DB *pgxpool.Pool

// InitDB initializes the database connection pool
func InitDB(pool *pgxpool.Pool) {
	DB = pool
}

// GetDB returns the database connection pool
func GetDB() *pgxpool.Pool {
	return DB
}

// WithTransaction executes a function within a database transaction
func WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		} else if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = fn(ctx)
	return err
}
