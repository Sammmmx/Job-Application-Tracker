package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Repository struct {
	DB *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		DB: db,
	}
}

const applicationColumns = `
    id, company, role, job_url, status,
    job_type, applied_at, interview_at,
    notes, created_at`

func (r *Repository) GetApplications(ctx context.Context) ([]Application, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT ${applicationColumns} FROM job_applications`,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var applications []Application

	for rows.Next() {
		var application Application
		if err := rows.Scan(
			&application.ID,
			&application.Company,
			&application.Role,
			&application.JobURL,
			&application.Status,
			&application.JobType,
			&application.AppliedAt,
			&application.InterviewAt,
			&application.Notes,
			&application.CreatedAt); err != nil {
			return nil, err
		}
		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}

func (r *Repository) CreateApplication(ctx context.Context, company, role, jobURL, status, jobType string, appliedAt, interviewAt time.Time, notes string) (Application, error) {
	var application Application
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO job_applications (company, role, job_url, status, job_type, applied_at, interview_at, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, company, role, job_url, status, job_type, applied_at, interview_at, notes, created_at`,
		company, role, jobURL, status, jobType, appliedAt, interviewAt, notes,
	).Scan(
		&application.ID,
		&application.Company,
		&application.Role,
		&application.JobURL,
		&application.Status,
		&application.JobType,
		&application.AppliedAt,
		&application.InterviewAt,
		&application.Notes,
		&application.CreatedAt,
	)
	if err != nil {
		return Application{}, err
	}

	return application, nil
}

func (r *Repository) UpdateApplication(ctx context.Context, id int, company, role, jobURL, status, jobType string, appliedAt, interviewAt time.Time, notes string) (Application, error) {
	var application Application
	err := r.DB.QueryRowContext(ctx,
		`UPDATE job_applications
		 SET company = $1, role = $2, job_url = $3, status = $4, job_type = $5, applied_at = $6, interview_at = $7, notes = $8
		 WHERE id = $9
		 RETURNING id, company, role, job_url, status, job_type, applied_at, interview_at, notes, created_at`,
		company, role, jobURL, status, jobType, appliedAt, interviewAt, notes, id,
	).Scan(
		&application.ID,
		&application.Company,
		&application.Role,
		&application.JobURL,
		&application.Status,
		&application.JobType,
		&application.AppliedAt,
		&application.InterviewAt,
		&application.Notes,
		&application.CreatedAt,
	)
	if err != nil {
		return Application{}, err
	}
	return application, nil
}

func (r *Repository) DeleteApplication(ctx context.Context, id int) error {
	result, err := r.DB.ExecContext(ctx,
		`DELETE FROM job_applications WHERE id = $1`,
		id,
	)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if count == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// Filter applications

func (r *Repository) FilterApplications(ctx context.Context, filter ApplicationFilter) ([]Application, error) {
	query := `SELECT ${applicationColumns} FROM job_applications WHERE 1=1`
	args := []any{}
	argIndex := 1

	if filter.ID != 0 {
		query += fmt.Sprintf(" AND id = $%d", argIndex)
		args = append(args, filter.ID)
		argIndex++
	}

	if filter.Company != "" {
		query += fmt.Sprintf(" AND company = $%d", argIndex)
		args = append(args, filter.Company)
		argIndex++
	}

	if filter.Role != "" {
		query += fmt.Sprintf(" AND role = $%d", argIndex)
		args = append(args, filter.Role)
		argIndex++
	}

	if filter.JobURL != "" {
		query += fmt.Sprintf(" AND job_url = $%d", argIndex)
		args = append(args, filter.JobURL)
		argIndex++
	}

	if filter.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.JobType != "" {
		query += fmt.Sprintf(" AND job_type = $%d", argIndex)
		args = append(args, filter.JobType)
		argIndex++
	}

	if !filter.StartDate.IsZero() && !filter.EndDate.IsZero() {
		args = append(args, filter.StartDate, filter.EndDate)

		query += fmt.Sprintf(
			" AND applied_at BETWEEN $%d AND $%d",
			len(args)-1,
			len(args),
		)
	}

	if !filter.StartDate.IsZero() &&
		filter.StartDate.After(filter.EndDate) {
		return nil, errors.New("start_date cannot be after end_date")
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var applications []Application

	for rows.Next() {
		var application Application
		if err := rows.Scan(
			&application.ID,
			&application.Company,
			&application.Role,
			&application.JobURL,
			&application.Status,
			&application.JobType,
			&application.AppliedAt,
			&application.InterviewAt,
			&application.Notes,
			&application.CreatedAt,
		); err != nil {
			return nil, err
		}
		applications = append(applications, application)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return applications, nil
}
