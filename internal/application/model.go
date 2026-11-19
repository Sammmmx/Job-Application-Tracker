package application

import (
	"encoding/json"
	"time"
)

type Application struct {
	ID          int        `json:"id"`
	Company     string     `json:"company"`
	Role        string     `json:"role"`
	JobURL      string     `json:"job_url"`
	Status      string     `json:"status"`
	JobType     string     `json:"job_type"`
	AppliedAt   time.Time  `json:"applied_at"`
	InterviewAt *time.Time `json:"interview_at"`
	Notes       string     `json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ApplicationInput is the full body of a POST or PUT.
type ApplicationInput struct {
	Company     string     `json:"company"`
	Role        string     `json:"role"`
	JobURL      string     `json:"job_url"`
	Status      string     `json:"status"`
	JobType     string     `json:"job_type"`
	AppliedAt   time.Time  `json:"applied_at"`
	InterviewAt *time.Time `json:"interview_at"`
	Notes       string     `json:"notes"`
}

// ApplicationPatch is the body of a PATCH. Every field is optional, and a nil
// pointer means "the client didn't send it, leave it alone".
type ApplicationPatch struct {
	Company     *string             `json:"company"`
	Role        *string             `json:"role"`
	JobURL      *string             `json:"job_url"`
	Status      *string             `json:"status"`
	JobType     *string             `json:"job_type"`
	AppliedAt   *time.Time          `json:"applied_at"`
	InterviewAt Nullable[time.Time] `json:"interview_at"`
	Notes       *string             `json:"notes"`
}

func (p ApplicationPatch) IsEmpty() bool {
	return p.Company == nil && p.Role == nil && p.JobURL == nil && p.Status == nil &&
		p.JobType == nil && p.AppliedAt == nil && !p.InterviewAt.Set && p.Notes == nil
}

// apply copies the fields the client sent onto in and leaves the rest untouched.
func (p ApplicationPatch) apply(in *ApplicationInput) {
	if p.Company != nil {
		in.Company = *p.Company
	}
	if p.Role != nil {
		in.Role = *p.Role
	}
	if p.JobURL != nil {
		in.JobURL = *p.JobURL
	}
	if p.Status != nil {
		in.Status = *p.Status
	}
	if p.JobType != nil {
		in.JobType = *p.JobType
	}
	if p.AppliedAt != nil {
		in.AppliedAt = *p.AppliedAt
	}
	if p.InterviewAt.Set {
		in.InterviewAt = p.InterviewAt.Value // nil here means "clear the interview date"
	}
	if p.Notes != nil {
		in.Notes = *p.Notes
	}
}

// Nullable tells apart the three things a JSON field can be:
//
//	absent         -> Set is false  (don't touch the column)
//	"field": null  -> Set is true,  Value is nil (set the column to NULL)
//	"field": x     -> Set is true,  Value points at x
//
// A plain *time.Time can't do this: absent and null both decode to nil.
type Nullable[T any] struct {
	Set   bool
	Value *T
}

func (n *Nullable[T]) UnmarshalJSON(data []byte) error {
	n.Set = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	n.Value = &v
	return nil
}

// Filled from query params, not JSON.
type ApplicationFilter struct {
	Company   string
	Role      string
	Status    string
	JobType   string
	StartDate time.Time
	EndDate   time.Time
}

// Page selects one slice of a list. A Limit of 0 means "use the default".
type Page struct {
	Limit  int
	Offset int
}

// ApplicationList is what GET /applications returns.
type ApplicationList struct {
	Items  []Application `json:"items"`
	Total  int           `json:"total"` // matching rows across all pages
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}
