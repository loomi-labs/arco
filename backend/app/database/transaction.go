package database

import (
	"context"
	"fmt"
	"github.com/loomi-labs/arco/backend/ent"
	"log/slog"
	"time"
)

// WithTx executes a function within a database transaction
func WithTx(ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) error) error {
	_, err := WithTxData(ctx, client, func(tx *ent.Tx) (struct{}, error) {
		return struct{}{}, fn(tx)
	})
	return err
}

// WithTxData executes a function within a database transaction and returns a value
func WithTxData[T any](ctx context.Context, client *ent.Client, fn func(tx *ent.Tx) (T, error)) (T, error) {
	startTime := time.Now()
	var data T

	tx, err := client.Tx(ctx)
	if err != nil {
		slog.Error("failed to begin database transaction", "error", err)
		return data, err
	}

	defer func() {
		if v := recover(); v != nil {
			if rerr := tx.Rollback(); rerr != nil {
				slog.Error("failed to rollback transaction during panic recovery", "panic", v, "rollback_error", rerr, "duration", time.Since(startTime))
			} else {
				slog.Warn("transaction rolled back due to panic", "panic", v, "duration", time.Since(startTime))
			}
			panic(v)
		}
	}()

	result, err := fn(tx)
	if err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			slog.Error("failed to rollback transaction after error", "original_error", err, "rollback_error", rerr, "duration", time.Since(startTime))
			err = fmt.Errorf("%w: rolling back transaction: %v", err, rerr)
		} else {
			slog.Debug("transaction rolled back due to error", "error", err, "duration", time.Since(startTime))
		}
		return data, err
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err, "duration", time.Since(startTime))
		return data, fmt.Errorf("committing transaction: %w", err)
	}

	slog.Debug("database transaction committed successfully", "duration", time.Since(startTime))
	return result, nil
}
