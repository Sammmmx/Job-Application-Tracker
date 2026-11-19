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

func strPtr(s string) *string { return &s }

// valid returns the input after normalize, which is what validate sees in real use.
func valid(mutate func(in *ApplicationInput)) ApplicationInput {
	in := validInput()
	mutate(&in)
	normalize(&in)
	return in
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
		{"whitespace-only company", func(in *ApplicationInput) { in.Company = "   " }, true},
		{"company too long", func(in *ApplicationInput) { in.Company = strings.Repeat("a", 151) }, true},
		{"company at limit", func(in *ApplicationInput) { in.Company = strings.Repeat("a", 150) }, false},
		{"150 multi-byte characters are fine", func(in *ApplicationInput) { in.Company = strings.Repeat("é", 150) }, false},
		{"151 multi-byte characters are too long", func(in *ApplicationInput) { in.Company = strings.Repeat("é", 151) }, true},
		{"unknown status", func(in *ApplicationInput) { in.Status = "Ghosted" }, true},
		{"status too long", func(in *ApplicationInput) { in.Status = strings.Repeat("a", 31) }, true},
		{"status in other case is accepted", func(in *ApplicationInput) { in.Status = "iNtErViEw" }, false},
		{"unknown job type", func(in *ApplicationInput) { in.JobType = "Hobby" }, true},
		{"job type may be empty", func(in *ApplicationInput) { in.JobType = "" }, false},
		{"job type in other case is accepted", func(in *ApplicationInput) { in.JobType = "  full-time " }, false},
		{"job url ok", func(in *ApplicationInput) { in.JobURL = "https://example.com/jobs/1" }, false},
		{"job url without scheme", func(in *ApplicationInput) { in.JobURL = "example.com/jobs" }, true},
		{"job url javascript", func(in *ApplicationInput) { in.JobURL = "javascript:alert(1)" }, true},
		{"job url ftp", func(in *ApplicationInput) { in.JobURL = "ftp://example.com/x" }, true},
		{"zero applied_at", func(in *ApplicationInput) {}, true}, // replaced below: validate itself must reject it
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
			in := valid(tc.mutate)
			if tc.name == "zero applied_at" {
				in.AppliedAt = time.Time{} // set after normalize, which would otherwise fill it in
			}

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

	t.Run("fixes the case of status and job type", func(t *testing.T) {
		in := ApplicationInput{Status: " offer ", JobType: "INTERNSHIP"}
		normalize(&in)
		if in.Status != "Offer" || in.JobType != "Internship" {
			t.Errorf("got status %q, job type %q", in.Status, in.JobType)
		}
	})

	t.Run("blank status becomes the default", func(t *testing.T) {
		in := ApplicationInput{Status: "   "}
		normalize(&in)
		if in.Status != "Applied" {
			t.Errorf("status = %q, want Applied", in.Status)
		}
	})

	t.Run("trims company and role", func(t *testing.T) {
		in := ApplicationInput{Company: "  Acme ", Role: "\tDev\n"}
		normalize(&in)
		if in.Company != "Acme" || in.Role != "Dev" {
			t.Errorf("got %q and %q", in.Company, in.Role)
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

	t.Run("saves the canonical spelling", func(t *testing.T) {
		svc := NewService(newFakeStore())
		in := validInput()
		in.Status, in.JobType = "interview", "full-time"
		got, err := svc.Create(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "Interview" || got.JobType != "Full-time" {
			t.Errorf("got %q / %q", got.Status, got.JobType)
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

func TestServiceUpdate(t *testing.T) {
	ctx := context.Background()

	t.Run("PUT without applied_at is rejected instead of overwriting the date", func(t *testing.T) {
		store := newFakeStore()
		svc := NewService(store)
		created, _ := svc.Create(ctx, validInput())

		in := validInput()
		in.AppliedAt = time.Time{}
		_, err := svc.Update(ctx, created.ID, in)

		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("expected ErrInvalid, got %v", err)
		}
		got, _ := svc.Get(ctx, created.ID)
		if !got.AppliedAt.Equal(validInput().AppliedAt) {
			t.Errorf("applied_at was changed to %v", got.AppliedAt)
		}
	})

	t.Run("replaces the record", func(t *testing.T) {
		svc := NewService(newFakeStore())
		created, _ := svc.Create(ctx, validInput())
		in := validInput()
		in.Status = "offer"
		got, err := svc.Update(ctx, created.ID, in)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "Offer" {
			t.Errorf("status = %q, want Offer", got.Status)
		}
	})
}

func TestServicePatch(t *testing.T) {
	ctx := context.Background()
	setup := func() (*Service, Application) {
		svc := NewService(newFakeStore())
		in := validInput()
		in.Notes = "original notes"
		in.JobType = "Internship"
		interview := in.AppliedAt.Add(72 * time.Hour)
		in.InterviewAt = &interview
		a, err := svc.Create(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		return svc, a
	}

	t.Run("changes only the fields that were sent", func(t *testing.T) {
		svc, a := setup()
		got, err := svc.Patch(ctx, a.ID, ApplicationPatch{Status: strPtr("Interview")})
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != "Interview" {
			t.Errorf("status = %q", got.Status)
		}
		if got.Company != "Acme" || got.Notes != "original notes" || got.JobType != "Internship" ||
			!got.AppliedAt.Equal(a.AppliedAt) || got.InterviewAt == nil {
			t.Errorf("untouched fields changed: %+v", got)
		}
	})

	t.Run("an empty string is a real value, not 'not sent'", func(t *testing.T) {
		svc, a := setup()
		got, err := svc.Patch(ctx, a.ID, ApplicationPatch{Notes: strPtr("")})
		if err != nil {
			t.Fatal(err)
		}
		if got.Notes != "" {
			t.Errorf("notes = %q, want empty", got.Notes)
		}
	})

	t.Run("null clears the interview date", func(t *testing.T) {
		svc, a := setup()
		got, err := svc.Patch(ctx, a.ID, ApplicationPatch{InterviewAt: Nullable[time.Time]{Set: true}})
		if err != nil {
			t.Fatal(err)
		}
		if got.InterviewAt != nil {
			t.Errorf("interview_at = %v, want nil", got.InterviewAt)
		}
	})

	t.Run("empty patch is rejected", func(t *testing.T) {
		svc, a := setup()
		if _, err := svc.Patch(ctx, a.ID, ApplicationPatch{}); !errors.Is(err, ErrInvalid) {
			t.Errorf("expected ErrInvalid, got %v", err)
		}
	})

	t.Run("result is validated like any other write", func(t *testing.T) {
		svc, a := setup()
		if _, err := svc.Patch(ctx, a.ID, ApplicationPatch{Status: strPtr("Ghosted")}); !errors.Is(err, ErrInvalid) {
			t.Errorf("bad status: expected ErrInvalid, got %v", err)
		}
		// Moving applied_at past the existing interview date must fail too.
		late := a.AppliedAt.Add(30 * 24 * time.Hour)
		if _, err := svc.Patch(ctx, a.ID, ApplicationPatch{AppliedAt: &late}); !errors.Is(err, ErrInvalid) {
			t.Errorf("applied_at after interview_at: expected ErrInvalid, got %v", err)
		}
	})

	t.Run("missing record", func(t *testing.T) {
		svc, _ := setup()
		if _, err := svc.Patch(ctx, 99, ApplicationPatch{Status: strPtr("Offer")}); !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
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
	if _, err := svc.List(context.Background(), f, Page{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("expected ErrInvalid, got %v", err)
	}
}

func TestServiceListPagination(t *testing.T) {
	ctx := context.Background()
	svc := NewService(newFakeStore())
	for i := 0; i < 5; i++ {
		if _, err := svc.Create(ctx, validInput()); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("defaults", func(t *testing.T) {
		got, err := svc.List(ctx, ApplicationFilter{}, Page{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Limit != DefaultLimit || got.Offset != 0 || got.Total != 5 || len(got.Items) != 5 {
			t.Errorf("got limit=%d offset=%d total=%d items=%d", got.Limit, got.Offset, got.Total, len(got.Items))
		}
	})

	t.Run("total is the full count, items are one page", func(t *testing.T) {
		got, _ := svc.List(ctx, ApplicationFilter{}, Page{Limit: 2, Offset: 2})
		if got.Total != 5 || len(got.Items) != 2 || got.Items[0].ID != 3 {
			t.Errorf("total=%d items=%d first=%d", got.Total, len(got.Items), got.Items[0].ID)
		}
	})

	t.Run("last page is shorter", func(t *testing.T) {
		got, _ := svc.List(ctx, ApplicationFilter{}, Page{Limit: 2, Offset: 4})
		if len(got.Items) != 1 {
			t.Errorf("items = %d, want 1", len(got.Items))
		}
	})

	t.Run("past the end is an empty page, not an error", func(t *testing.T) {
		got, err := svc.List(ctx, ApplicationFilter{}, Page{Limit: 2, Offset: 50})
		if err != nil || len(got.Items) != 0 || got.Total != 5 {
			t.Errorf("err=%v items=%d total=%d", err, len(got.Items), got.Total)
		}
	})

	for name, p := range map[string]Page{
		"limit above the maximum": {Limit: MaxLimit + 1},
		"negative limit":          {Limit: -1},
		"negative offset":         {Limit: 10, Offset: -1},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := svc.List(ctx, ApplicationFilter{}, p); !errors.Is(err, ErrInvalid) {
				t.Errorf("expected ErrInvalid, got %v", err)
			}
		})
	}
}
