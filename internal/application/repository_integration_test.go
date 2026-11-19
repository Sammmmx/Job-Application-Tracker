package application

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// testRepo connects to the test database and empties the table.
// Tests are skipped when TEST_DATABASE_URL is not set, so plain
// `go test ./...` still works without a database.
func testRepo(t *testing.T) *Repository {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec("TRUNCATE job_applications RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate (have all migrations been applied?): %v", err)
	}
	return NewRepository(db)
}

func day(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func seed(t *testing.T, r *Repository, in ApplicationInput) Application {
	t.Helper()
	a, err := r.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return a
}

func listAll(t *testing.T, r *Repository, f ApplicationFilter) []Application {
	t.Helper()
	got, _, err := r.List(context.Background(), f, Page{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestRepoCreateAndGet(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()
	interview := time.Date(2026, 11, 10, 14, 30, 0, 0, time.UTC)

	created, err := r.Create(ctx, ApplicationInput{
		Company: "Acme", Role: "Backend Dev", Status: "Applied",
		AppliedAt: day(2026, 11, 1), InterviewAt: &interview,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.CreatedAt.IsZero() {
		t.Errorf("unexpected created record: %+v", created)
	}

	got, err := r.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Company != "Acme" || got.Role != "Backend Dev" {
		t.Errorf("round trip mismatch: %+v", got)
	}
	if got.AppliedAt.Format("2006-01-02") != "2026-11-01" {
		t.Errorf("applied_at = %v, want 2026-11-01", got.AppliedAt)
	}
	if got.InterviewAt == nil || !got.InterviewAt.Equal(interview) {
		t.Errorf("interview_at = %v, want %v", got.InterviewAt, interview)
	}
}

// Needs migration 000004. With the old TIMESTAMP column the +05:30 offset was thrown
// away, so 14:30+05:30 came back as 14:30 UTC, which is five and a half hours wrong.
func TestRepoInterviewAtKeepsTheRealInstant(t *testing.T) {
	r := testRepo(t)
	ist := time.FixedZone("IST", 5*3600+30*60)
	interview := time.Date(2026, 11, 10, 14, 30, 0, 0, ist)

	a := seed(t, r, ApplicationInput{
		Company: "Acme", Role: "Dev", Status: "Applied",
		AppliedAt: day(2026, 11, 1), InterviewAt: &interview,
	})
	got, err := r.Get(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.InterviewAt == nil || !got.InterviewAt.Equal(interview) {
		t.Errorf("interview_at = %v, want the same instant as %v", got.InterviewAt, interview)
	}
}

func TestRepoNullInterviewAt(t *testing.T) {
	r := testRepo(t)
	a := seed(t, r, ApplicationInput{
		Company: "Acme", Role: "Dev", Status: "Applied", AppliedAt: day(2026, 11, 1),
	})
	got, err := r.Get(context.Background(), a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.InterviewAt != nil {
		t.Errorf("interview_at = %v, want nil (NULL)", got.InterviewAt)
	}
}

func TestRepoGetMissing(t *testing.T) {
	r := testRepo(t)
	_, err := r.Get(context.Background(), 999)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestRepoListFilters(t *testing.T) {
	r := testRepo(t)
	seed(t, r, ApplicationInput{Company: "Acme Corp", Role: "Backend Dev", Status: "Applied", JobType: "Full-time", AppliedAt: day(2026, 10, 1)})
	seed(t, r, ApplicationInput{Company: "Globex", Role: "Frontend Dev", Status: "Interview", JobType: "Internship", AppliedAt: day(2026, 11, 1)})
	seed(t, r, ApplicationInput{Company: "Initech", Role: "Backend Intern", Status: "Rejected", JobType: "Internship", AppliedAt: day(2026, 11, 15)})
	seed(t, r, ApplicationInput{Company: "100% Remote Inc", Role: "SRE", Status: "Applied", AppliedAt: day(2026, 9, 1)})
	seed(t, r, ApplicationInput{Company: "A_B Labs", Role: "SRE", Status: "Applied", AppliedAt: day(2026, 9, 2)})

	tests := []struct {
		name   string
		filter ApplicationFilter
		want   int
	}{
		{"no filter", ApplicationFilter{}, 5},
		{"company substring, any case", ApplicationFilter{Company: "acme"}, 1},
		{"role substring", ApplicationFilter{Role: "backend"}, 2},
		{"status is case-insensitive", ApplicationFilter{Status: "applied"}, 3},
		{"job type", ApplicationFilter{JobType: "internship"}, 2},
		{"start date only", ApplicationFilter{StartDate: day(2026, 11, 1)}, 2},
		{"end date only", ApplicationFilter{EndDate: day(2026, 10, 31)}, 3},
		{"date range", ApplicationFilter{StartDate: day(2026, 10, 15), EndDate: day(2026, 11, 5)}, 1},
		{"combined filters", ApplicationFilter{Role: "backend", JobType: "internship"}, 1},
		{"no match", ApplicationFilter{Company: "nonexistent"}, 0},
		{"percent sign is literal, not a wildcard", ApplicationFilter{Company: "100%"}, 1},
		{"a lone percent sign matches only names containing it", ApplicationFilter{Company: "%"}, 1},
		{"underscore is literal, not a wildcard", ApplicationFilter{Company: "a_b"}, 1},
		{"underscore does not match any single character", ApplicationFilter{Company: "Gl_bex"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := listAll(t, r, tc.filter); len(got) != tc.want {
				t.Errorf("got %d results, want %d", len(got), tc.want)
			}
		})
	}

	t.Run("ordered newest first", func(t *testing.T) {
		got := listAll(t, r, ApplicationFilter{})
		if got[0].Company != "Initech" || got[len(got)-1].Company != "100% Remote Inc" {
			t.Errorf("unexpected order: first %s, last %s", got[0].Company, got[len(got)-1].Company)
		}
	})
}

func TestRepoListPagination(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()
	for i := 1; i <= 7; i++ { // applied on Nov 1..7, so newest first is Nov 7 down to Nov 1
		seed(t, r, ApplicationInput{Company: "Co", Role: "Dev", Status: "Applied", AppliedAt: day(2026, 11, i)})
	}

	page1, total, err := r.List(ctx, ApplicationFilter{}, Page{Limit: 3, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if total != 7 || len(page1) != 3 || page1[0].AppliedAt.Day() != 7 || page1[2].AppliedAt.Day() != 5 {
		t.Errorf("page 1: total=%d len=%d", total, len(page1))
	}

	page3, total, _ := r.List(ctx, ApplicationFilter{}, Page{Limit: 3, Offset: 6})
	if total != 7 || len(page3) != 1 || page3[0].AppliedAt.Day() != 1 {
		t.Errorf("last page: total=%d len=%d", total, len(page3))
	}

	beyond, total, err := r.List(ctx, ApplicationFilter{}, Page{Limit: 3, Offset: 70})
	if err != nil || total != 7 || len(beyond) != 0 || beyond == nil {
		t.Errorf("beyond the end: err=%v total=%d items=%v", err, total, beyond)
	}

	t.Run("total counts the filtered rows, not the whole table", func(t *testing.T) {
		_, total, _ := r.List(ctx, ApplicationFilter{StartDate: day(2026, 11, 5)}, Page{Limit: 1})
		if total != 3 {
			t.Errorf("total = %d, want 3", total)
		}
	})
}

func TestRepoUpdate(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()
	a := seed(t, r, ApplicationInput{Company: "Acme", Role: "Dev", Status: "Applied", AppliedAt: day(2026, 11, 1)})

	updated, err := r.Update(ctx, a.ID, ApplicationInput{
		Company: "Acme", Role: "Dev", Status: "Interview", AppliedAt: day(2026, 11, 1), Notes: "phone screen",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "Interview" || updated.Notes != "phone screen" {
		t.Errorf("update not applied: %+v", updated)
	}

	_, err = r.Update(ctx, 999, ApplicationInput{Company: "X", Role: "Y", Status: "Applied", AppliedAt: day(2026, 11, 1)})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("update missing: expected sql.ErrNoRows, got %v", err)
	}
}

func TestRepoDelete(t *testing.T) {
	r := testRepo(t)
	ctx := context.Background()
	a := seed(t, r, ApplicationInput{Company: "Acme", Role: "Dev", Status: "Applied", AppliedAt: day(2026, 11, 1)})

	if err := r.Delete(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, a.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("second delete: expected sql.ErrNoRows, got %v", err)
	}
}

// The whole stack except HTTP: real service on the real repository.
func TestServiceOnRealRepository(t *testing.T) {
	r := testRepo(t)
	svc := NewService(r)
	ctx := context.Background()

	created, err := svc.Create(ctx, ApplicationInput{
		Company: "  Acme ", Role: "Dev", Status: "interview", JobType: "internship",
		AppliedAt: day(2026, 11, 1), Notes: "first note",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Company != "Acme" || created.Status != "Interview" || created.JobType != "Internship" {
		t.Errorf("not normalised before saving: %+v", created)
	}

	patched, err := svc.Patch(ctx, created.ID, ApplicationPatch{Status: strPtr("Offer")})
	if err != nil {
		t.Fatal(err)
	}
	if patched.Status != "Offer" || patched.Notes != "first note" || patched.AppliedAt.Format("2006-01-02") != "2026-11-01" {
		t.Errorf("patch changed more than the status: %+v", patched)
	}

	if _, err := svc.Get(ctx, 999); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
