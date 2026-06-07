package ingest

import (
	"context"

	"github.com/ilcm96/codex-usage/internal/server/ingeststore"
)

type Service struct {
	repository PostgresRepository
	rawDir     string
}

func NewService(repository PostgresRepository, rawDir string) Service {
	return Service{repository: repository, rawDir: rawDir}
}

func (s Service) StoreRaw(ctx context.Context, metadata ingeststore.Metadata, path string, rawSizeBytes int64) (ingeststore.Result, error) {
	return s.repository.StoreRaw(ctx, metadata, path, rawSizeBytes)
}

func (s Service) StoreRawBatch(ctx context.Context, inputs []ingeststore.RawInput) ([]ingeststore.Result, error) {
	return s.repository.StoreRawBatch(ctx, inputs)
}
