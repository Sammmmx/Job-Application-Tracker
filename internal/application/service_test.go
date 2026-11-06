package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func validInput() ApplicationInput {
	return ApplicationInput{
		Company:   "Acme",
		Role:      "Backend Developer",
		AppliedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(in *ApplicationInput)
		wantErr bool
	}{
		{"valid input", func(in *ApplicationInput) {}, false},
		{"missing company", func(in *ApplicationInput) { in.Company = "" }, true},
		{"missing role", func(in *ApplicationInput) { in.Role = "" }, true},
		{"company too long", func(in *ApplicationInput) { in.Company = strings.Repeat("a", 151) }, true},
		{"company at limit", func(in *ApplicationInput) { in.Company = strings.Repeat("a", 150) }, false},
		{"status too long", func(in *ApplicationInput) { in.Status = strings.Repeat("a", 31) }, true},
		{"job type too long", func(in *ApplicationInput) { in.JobType = strings.Repeat("a", 31) }, true},
		{"zero applied_at", func(in *ApplicationInput) { in.AppliedAt = time.Time{} }, true},
		{"interview before applied", func(in *ApplicationInput) {
			earlier := in.AppliedAt.Add(-time.Hour)
			in.InterviewAt = &earlier
		}, true},
		{"interview after applied", func(in *ApplicationInput) {
			later := in.AppliedAt.Add(24 * time.Hour)
			in.InterviewAt = &later
		}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := validInput()
			tc.mutate(&in)

			err := validate(in)

			if tc.wantErr {
				if !errors.Is(err, ErrInvalid) {
					t.Fatalf("expected ErrInvalid, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	t.Run("fills default status and applied_at", func(t *testing.T) {
		in := ApplicationInput{}
		normalize(&in)
		if in.Status != "Applied" {
			t.Errorf("status = %q, want %q", in.Status, "Applied")
		}
		if in.AppliedAt.IsZero() {
			t.Error("applied_at should have been set")
		}
	})

	t.Run("keeps provided values", func(t *testing.T) {
		when := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
		in := ApplicationInput{Status: "Interview", AppliedAt: when}
		normalize(&in)
		if in.Status != "Interview" {
			t.Errorf("status = %q, want Interview", in.Status)
		}
		if !in.AppliedAt.Equal(when) {
			t.Errorf("applied_at changed to %v", in.AppliedAt)
		}
	})
}

func TestServiceCreate(t *testing.T) {
	ctx := context.Background()

	t.Run("applies default status", func(t *testing.T) {
		svc := NewService(newFakeStore())
		got, err := svc.Create(ctx, validInput())
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "Applied" {
			t.Errorf("status = %q, want Applied", got.Status)
		}
		if got.ID == 0 {
			t.Error("expected an ID to be assigned")
		}
	})

	t.Run("invalid input never reaches the store", func(t *testing.T) {
		store := newFakeStore()
		svc := NewService(store)
		in := validInput()
		in.Company = ""

		_, err := svc.Create(ctx, in)

		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid, got %v", err)
		}
		if store.writes != 0 {
			t.Errorf("store was written %d times, want 0", store.writes)
		}
	})
}

func TestServiceNotFound(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeStore())

	if _, err := svc.Get(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := svc.Update(ctx, 99, validInput()); !errors.Is(err, ErrNotFound) {
		t.Errorf("Update: expected ErrNotFound, got %v", err)
	}
	if err := svc.Delete(ctx, 99); !errors.Is(err, ErrNotFound) {
		t.Errorf("Delete: expected ErrNotFound, got %v", err)
	}
}

func TestServiceListDateRange(t *testing.T) {
	svc := NewService(newFakeStore())
	f := ApplicationFilter{
		StartDate: time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
	}
	if _, err := svc.List(context.Background(), f); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}
