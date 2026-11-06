package application

import "time"

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

// Filled from query params, not JSON.
type ApplicationFilter struct {
	Company   string
	Role      string
	Status    string
	JobType   string
	StartDate time.Time
	EndDate   time.Time
}
