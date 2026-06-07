package sessions

import (
	"context"
	"errors"
)

var (
	ErrNotFound        = errors.New("session not found")
	ErrUnsupportedSort = errors.New("unsupported sort")
	ErrUnsupportedKind = errors.New("unsupported kind")
)

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) ListSimple(ctx context.Context, params ListParams) ([]SimpleSession, error) {
	params.Limit = clampLimit(params.Limit, 50, 500)
	if _, ok := sessionOrderBy(params.Sort); !ok {
		return nil, ErrUnsupportedSort
	}
	return s.repository.ListSimple(ctx, params)
}

func (s Service) List(ctx context.Context, params ListParams) (ListResult, error) {
	params.Limit = clampLimit(params.Limit, 50, 200)
	if params.Offset < 0 {
		params.Offset = 0
	}
	if _, ok := sessionOrderBy(params.Sort); !ok {
		return ListResult{}, ErrUnsupportedSort
	}
	return s.repository.List(ctx, params)
}

func (s Service) ListItems(ctx context.Context, params ListParams) ([]ListItem, error) {
	params.Limit = clampLimit(params.Limit, 50, 200)
	if params.Offset < 0 {
		params.Offset = 0
	}
	if _, ok := sessionOrderBy(params.Sort); !ok {
		return nil, ErrUnsupportedSort
	}
	return s.repository.ListItems(ctx, params)
}

func (s Service) Detail(ctx context.Context, id string) (DetailResult, error) {
	return s.repository.Detail(ctx, id)
}

func (s Service) Reader(ctx context.Context, id string, params ReaderParams) (ReaderResult, error) {
	params.Limit = clampLimit(params.Limit, 30, 100)
	if params.Offset < 0 {
		params.Offset = 0
	}
	return s.repository.Reader(ctx, id, params)
}

func (s Service) Timeline(ctx context.Context, id string, params TimelineParams) (TimelineResult, error) {
	params.Limit = clampLimit(params.Limit, 100, 300)
	if params.Offset < 0 {
		params.Offset = 0
	}
	switch params.Kind {
	case "", "message", "tool":
	default:
		return TimelineResult{}, ErrUnsupportedKind
	}
	return s.repository.Timeline(ctx, id, params)
}

func clampLimit(value int, fallback int, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}
