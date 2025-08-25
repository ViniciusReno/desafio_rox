package domain

import "context"

type Repository interface {
	DayExists(ctx context.Context, day string) (bool, error)
	InsertBatch(ctx context.Context, day string, lines []string) error
}

type Service struct {
	repo Repository
}

func NewService(r Repository) *Service {
	return &Service{repo: r}
}

func (s *Service) Import(ctx context.Context, day string, lines []string) error {
	exists, err := s.repo.DayExists(ctx, day)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.repo.InsertBatch(ctx, day, lines)
}
