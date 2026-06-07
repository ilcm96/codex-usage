package archive

import "context"

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) Status(ctx context.Context) (Status, error) {
	return s.repository.Status(ctx)
}

func (s Service) Health(ctx context.Context) (Health, error) {
	return s.repository.Health(ctx)
}

func (s Service) ByDevice(ctx context.Context) ([]DeviceSummary, error) {
	return s.repository.ByDevice(ctx)
}

func (s Service) ByRepository(ctx context.Context, limit int) ([]RepositorySummary, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repository.ByRepository(ctx, limit)
}

func (s Service) Integrity(ctx context.Context) (IntegrityResult, error) {
	return s.repository.Integrity(ctx)
}
