package projects

import "context"

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) ListRepositories(ctx context.Context) ([]RepositorySummary, error) {
	return s.repository.ListRepositories(ctx)
}

func (s Service) ListProjects(ctx context.Context) ([]ProjectSummary, error) {
	return s.repository.ListProjects(ctx)
}
