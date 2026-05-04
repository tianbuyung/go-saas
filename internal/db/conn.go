package db

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(dbPoolURL string) *pgxpool.Pool {
	cfg, err := pgxpool.ParseConfig(dbPoolURL)
	if err != nil {
		log.Fatal(err)
	}

	// REQUIRED for pgBouncer
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	return pool
}
