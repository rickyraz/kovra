package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// WithTx executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// Otherwise, the transaction is committed.
func (db *DB) WithTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// WithTxResult executes a function within a transaction and returns a result.
func WithTxResult[T any](ctx context.Context, db *DB, fn func(tx pgx.Tx) (T, error)) (T, error) {
	var result T

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return result, fmt.Errorf("begin transaction: %w", err)
	}

	result, err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return result, fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
		}
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}
