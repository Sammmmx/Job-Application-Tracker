package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrNotFound = errors.New("application not found")
	ErrInvalid  = errors.New("invalid application")
)

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

// The accepted values. To allow a new one, add it here and nowhere else.
var (
	allowedStatuses = []string{"Applied", "Interview", "Offer", "Rejected", "Withdrawn"}
	allowedJobTypes = []string{"Full-time", "Part-time", "Internship", "Contract"}
)

type Store interface {
	List(ctx context.Context, f ApplicationFilter, p Page) ([]Application, int, error)
	Get(ctx context.Context, id int) (Application, error)
	Create(ctx context.Context, in ApplicationInput) (Application, error)
	Update(ctx context.Context, id int, in ApplicationInput) (Application, error)
	Delete(ctx context.Context, id int) error
}

var _ Store = (*Repository)(nil) // compile-time check that Repository satisfies Store

type Service struct{ repo Store }

func NewService(r Store) *Service { return &Service{repo: r} }

// canonical matches value against the allowed list ignoring case and surrounding
// spaces, and returns the official spelling ("full-time " becomes "Full-time").
func canonical(value string, allowed []string) (string, bool) {
	v := strings.TrimSpace(value)
	for _, a := range allowed {
		if strings.EqualFold(a, v) {
			return a, true
		}
	}
	return v, false
}

// normalize cleans the input and fills in defaults before validation and saving.
func normalize(in *ApplicationInput) {
	in.Company = strings.TrimSpace(in.Company)
	in.Role = strings.TrimSpace(in.Role)
	in.JobURL = strings.TrimSpace(in.JobURL)

	if strings.TrimSpace(in.Status) == "" {
		in.Status = "Applied"
	}
	in.Status, _ = canonical(in.Status, allowedStatuses)
	in.JobType, _ = canonical(in.JobType, allowedJobTypes)

	if in.AppliedAt.IsZero() {
		in.AppliedAt = time.Now().UTC()
	}
}

func validate(in ApplicationInput) error {
	if in.Company == "" || in.Role == "" {
		return fmt.Errorf("%w: company and role are required", ErrInvalid)
	}
	// Count characters, not bytes: the database limit is in characters, and a
	// name in Hindi or Japanese takes 3 bytes per character.
	if utf8.RuneCountInString(in.Company) > 150 || utf8.RuneCountInString(in.Role) > 150 {
		return fmt.Errorf("%w: company and role must be at most 150 characters", ErrInvalid)
	}
	if !contains(allowedStatuses, in.Status) {
		return fmt.Errorf("%w: status must be one of %s", ErrInvalid, strings.Join(allowedStatuses, ", "))
	}
	if in.JobType != "" && !contains(allowedJobTypes, in.JobType) {
		return fmt.Errorf("%w: job_type must be one of %s", ErrInvalid, strings.Join(allowedJobTypes, ", "))
	}
	if in.JobURL != "" {
		u, err := url.ParseRequestURI(in.JobURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("%w: job_url must be a valid http or https URL", ErrInvalid)
		}
	}
	if in.AppliedAt.IsZero() {
		return fmt.Errorf("%w: applied_at is required", ErrInvalid)
	}
	if in.InterviewAt != nil && in.InterviewAt.Before(in.AppliedAt) {
		return fmt.Errorf("%w: interview_at cannot be before applied_at", ErrInvalid)
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func notFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func (s *Service) List(ctx context.Context, f ApplicationFilter, p Page) (ApplicationList, error) {
	if !f.StartDate.IsZero() && !f.EndDate.IsZero() && f.StartDate.After(f.EndDate) {
		return ApplicationList{}, fmt.Errorf("%w: start_date cannot be after end_date", ErrInvalid)
	}
	if p.Limit == 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit < 1 || p.Limit > MaxLimit {
		return ApplicationList{}, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalid, MaxLimit)
	}
	if p.Offset < 0 {
		return ApplicationList{}, fmt.Errorf("%w: offset cannot be negative", ErrInvalid)
	}

	items, total, err := s.repo.List(ctx, f, p)
	if err != nil {
		return ApplicationList{}, err
	}
	return ApplicationList{Items: items, Total: total, Limit: p.Limit, Offset: p.Offset}, nil
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

// Update is PUT: it replaces the whole record, so the client must send everything.
// applied_at is not defaulted here, because a forgotten field would silently
// overwrite the real date with today's. Use Patch to change only some fields.
func (s *Service) Update(ctx context.Context, id int, in ApplicationInput) (Application, error) {
	if in.AppliedAt.IsZero() {
		return Application{}, fmt.Errorf("%w: applied_at is required for PUT (use PATCH to change only some fields)", ErrInvalid)
	}
	normalize(&in)
	if err := validate(in); err != nil {
		return Application{}, err
	}
	a, err := s.repo.Update(ctx, id, in)
	return a, notFound(err)
}

// Patch changes only the fields the client sent: load the record, overlay the
// changes, then run the same checks as any other write.
func (s *Service) Patch(ctx context.Context, id int, p ApplicationPatch) (Application, error) {
	if p.IsEmpty() {
		return Application{}, fmt.Errorf("%w: no fields to update", ErrInvalid)
	}
	cur, err := s.repo.Get(ctx, id)
	if err != nil {
		return Application{}, notFound(err)
	}
	in := ApplicationInput{
		Company: cur.Company, Role: cur.Role, JobURL: cur.JobURL,
		Status: cur.Status, JobType: cur.JobType, AppliedAt: cur.AppliedAt,
		InterviewAt: cur.InterviewAt, Notes: cur.Notes,
	}
	p.apply(&in)
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
