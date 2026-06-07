package filteroptions

import "context"

type Service struct {
	repository PostgresRepository
}

func NewService(repository PostgresRepository) Service {
	return Service{repository: repository}
}

func (s Service) List(ctx context.Context) (Result, error) {
	return s.repository.List(ctx)
}
