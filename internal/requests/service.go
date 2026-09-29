package requests

import (
	"context"
)

// Service provides high-level business operations for HTTP transactions.
type Service struct {
	repo Repository
}

// NewService creates a new Service instance.
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Save records a transaction in the repository.
func (s *Service) Save(ctx context.Context, tx *HTTPTransaction) error {
	return s.repo.Save(ctx, tx)
}

// GetByID retrieves a transaction by its ID.
func (s *Service) GetByID(ctx context.Context, id string) (*HTTPTransaction, error) {
	return s.repo.GetByID(ctx, id)
}

// List returns a paginated list of request summaries matching the filter.
func (s *Service) List(ctx context.Context, filter RequestFilter) ([]RequestSummary, int, error) {
	return s.repo.List(ctx, filter)
}

// DeleteByID removes a transaction by ID.
func (s *Service) DeleteByID(ctx context.Context, id string) error {
	return s.repo.DeleteByID(ctx, id)
}

// Clear removes all stored transactions.
func (s *Service) Clear(ctx context.Context) error {
	return s.repo.Clear(ctx)
}
