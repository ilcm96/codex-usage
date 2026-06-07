package usage

import (
	"context"
	"errors"
)

var (
	ErrUnsupportedBucket = errors.New("unsupported bucket")
	ErrUnsupportedGroup  = errors.New("unsupported group")
)

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) Totals(ctx context.Context) (GlobalTotals, error) {
	return s.repository.Totals(ctx)
}

func (s Service) Windows(ctx context.Context) ([]Window, error) {
	return s.repository.Windows(ctx)
}

func (s Service) Series(ctx context.Context, params SeriesParams) ([]SeriesBucket, error) {
	switch params.Bucket {
	case "":
		params.Bucket = "day"
	case "day", "week", "month":
	default:
		return nil, ErrUnsupportedBucket
	}
	return s.repository.Series(ctx, params)
}

func (s Service) Breakdown(ctx context.Context, params BreakdownParams) ([]BreakdownItem, error) {
	if params.GroupBy == "" {
		params.GroupBy = "model"
	}
	switch params.GroupBy {
	case "model", "repository", "project", "device", "language":
	default:
		return nil, ErrUnsupportedGroup
	}
	if params.Limit <= 0 {
		params.Limit = 12
	}
	if params.Limit > 50 {
		params.Limit = 50
	}
	return s.repository.Breakdown(ctx, params)
}

func (s Service) Summary(ctx context.Context, filters Filters) (Summary, error) {
	current, activeDays, err := s.repository.Summary(ctx, filters)
	if err != nil {
		return Summary{}, err
	}

	cacheHitRate := 0.0
	if current.InputTokens > 0 {
		cacheHitRate = float64(current.CachedInputTokens) / float64(current.InputTokens)
	}
	avgSessionCost := 0.0
	if current.Sessions > 0 {
		avgSessionCost = current.CostUSD / float64(current.Sessions)
	}

	return Summary{
		Current:        current,
		ActiveDays:     activeDays,
		CacheHitRate:   cacheHitRate,
		AvgSessionCost: avgSessionCost,
	}, nil
}

func (s Service) Calendar(ctx context.Context, params CalendarParams) ([]CalendarDay, error) {
	if params.Days <= 0 {
		params.Days = 120
	}
	if params.Days > 366 {
		params.Days = 366
	}
	return s.repository.Calendar(ctx, params)
}
