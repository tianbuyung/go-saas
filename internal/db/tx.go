package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	WithTx(ctx context.Context, fn func(*Queries) error) error
}

type pgxTxManager struct {
	pool *pgxpool.Pool
}

func NewTxManager(pool *pgxpool.Pool) TxManager {
	return &pgxTxManager{pool: pool}
}

func (m *pgxTxManager) WithTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := m.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	if err := fn(New(tx)); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	return tx.Commit(ctx)
}
