package search

import (
	"context"
	"errors"
	"strings"
)

var ErrUnsupportedKind = errors.New("unsupported kind")

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) Search(ctx context.Context, params Params) (SearchResult, error) {
	params.Query = strings.TrimSpace(params.Query)
	if params.Query == "" {
		return SearchResult{Items: []Result{}, Total: 0, NextOffset: 0, Offset: 0, TotalKnown: true}, nil
	}
	if params.Limit <= 0 {
		params.Limit = 50
	}
	if params.Limit > 200 {
		params.Limit = 200
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
	switch params.Kind {
	case "", "message", "user", "assistant", "tool", "all":
	default:
		return SearchResult{}, ErrUnsupportedKind
	}
	return s.repository.Search(ctx, params)
}
