package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrNotFound = errors.New("application not found")
	ErrInvalid  = errors.New("invalid application")
)

type Store interface {
	List(ctx context.Context, f ApplicationFilter) ([]Application, error)
	Get(ctx context.Context, id int) (Application, error)
	Create(ctx context.Context, in ApplicationInput) (Application, error)
	Update(ctx context.Context, id int, in ApplicationInput) (Application, error)
	Delete(ctx context.Context, id int) error
}

var _ Store = (*Repository)(nil) // compile-time check that Repository satisfies Store

type Service struct{ repo Store }

func NewService(r Store) *Service { return &Service{repo: r} }

// normalize fills in defaults before validation and saving.
func normalize(in *ApplicationInput) {
	if in.Status == "" {
		in.Status = "Applied"
	}

	if in.AppliedAt.IsZero() {
		in.AppliedAt = time.Now()
	}
}

func validate(in ApplicationInput) error {
	if in.Company == "" || in.Role == "" {
		return fmt.Errorf("%w: company and role are required", ErrInvalid)
	}
	if len(in.Company) > 150 || len(in.Role) > 150 {
		return fmt.Errorf("%w: company and role must be at most 150 characters", ErrInvalid)
	}
	if len(in.Status) > 30 || len(in.JobType) > 30 {
		return fmt.Errorf("%w: status and job_type must be at most 30 characters", ErrInvalid)
	}
	if in.AppliedAt.IsZero() {
		return fmt.Errorf("%w: applied_at is required", ErrInvalid)
	}
	if in.InterviewAt != nil && in.InterviewAt.Before(in.AppliedAt) {
		return fmt.Errorf("%w: interview_at cannot be before applied_at", ErrInvalid)
	}
	return nil
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Service) List(ctx context.Context, f ApplicationFilter) ([]Application, error) {
	if !f.StartDate.IsZero() && !f.EndDate.IsZero() && f.StartDate.After(f.EndDate) {
		return nil, fmt.Errorf("%w: start_date cannot be after end_date", ErrInvalid)
	}
	return s.repo.List(ctx, f)
}

func (s *Service) Get(ctx context.Context, id int) (Application, error) {
	a, err := s.repo.Get(ctx, id)
	return a, notFound(err)
}

func (s *Service) Create(ctx context.Context, in ApplicationInput) (Application, error) {
	normalize(&in)
	if err := validate(in); err != nil {
		return Application{}, err
	}
	return s.repo.Create(ctx, in)
}

func (s *Service) Update(ctx context.Context, id int, in ApplicationInput) (Application, error) {
	normalize(&in)
	if err := validate(in); err != nil {
		return Application{}, err
	}
	a, err := s.repo.Update(ctx, id, in)
	return a, notFound(err)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return notFound(s.repo.Delete(ctx, id))
}
