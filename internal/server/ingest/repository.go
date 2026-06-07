package ingest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ilcm96/codex-usage/internal/server/ingeststore"
)

type PostgresRepository struct {
	store ingeststore.Store
}

func NewPostgresRepository(db *pgxpool.Pool) PostgresRepository {
	return PostgresRepository{store: ingeststore.New(db)}
}

func (r PostgresRepository) StoreRaw(ctx context.Context, metadata ingeststore.Metadata, path string, rawSizeBytes int64) (ingeststore.Result, error) {
	return r.store.StoreRaw(ctx, metadata, path, rawSizeBytes)
}

func (r PostgresRepository) StoreRawBatch(ctx context.Context, inputs []ingeststore.RawInput) ([]ingeststore.Result, error) {
	return r.store.StoreRawBatch(ctx, inputs)
}
