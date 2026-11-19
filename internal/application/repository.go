package application

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type Repository struct{ DB *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{DB: db} }

const applicationColumns = `id, company, role, job_url, status,
	job_type, applied_at, interview_at, notes, created_at`

type scanner interface{ Scan(dest ...any) error }

func scanApplication(s scanner) (Application, error) {
	var a Application
	err := s.Scan(&a.ID, &a.Company, &a.Role, &a.JobURL, &a.Status,
		&a.JobType, &a.AppliedAt, &a.InterviewAt, &a.Notes, &a.CreatedAt)
	return a, err
}

// buildWhere turns the filter into a WHERE clause plus its $1, $2... arguments.
// Only placeholders are ever put into the SQL text, never the user's values.
func buildWhere(f ApplicationFilter) (string, []any) {
	where := " WHERE 1=1"
	args := []any{}

	add := func(cond string, v any) {
		args = append(args, v)
		where += fmt.Sprintf(" AND "+cond, len(args))
	}
	if f.Company != "" {
		add("company ILIKE $%d", "%"+escapeLike(f.Company)+"%")
	}
	if f.Role != "" {
		add("role ILIKE $%d", "%"+escapeLike(f.Role)+"%")
	}
	if f.Status != "" {
		add("lower(status) = lower($%d)", f.Status)
	}
	if f.JobType != "" {
		add("lower(job_type) = lower($%d)", f.JobType)
	}
	if !f.StartDate.IsZero() {
		add("applied_at >= $%d", f.StartDate)
	}
	if !f.EndDate.IsZero() {
		add("applied_at <= $%d", f.EndDate)
	}
	return where, args
}

// escapeLike makes %, _ and \ in the user's text match literally. Without this,
// searching for "100%" or "a_b" would treat the characters as wildcards.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// List returns one page of matching rows, plus the total number of matches.
func (r *Repository) List(ctx context.Context, f ApplicationFilter, p Page) ([]Application, int, error) {
	where, args := buildWhere(f)

	var total int
	if err := r.DB.QueryRowContext(ctx,
		"SELECT count(*) FROM job_applications"+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT " + applicationColumns + " FROM job_applications" + where +
		" ORDER BY applied_at DESC, id DESC" +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, p.Limit, p.Offset)

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	apps := []Application{} // empty slice -> JSON [] instead of null
	for rows.Next() {
		a, err := scanApplication(rows)
		if err != nil {
			return nil, 0, err
		}
		apps = append(apps, a)
	}
	return apps, total, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int) (Application, error) {
	return scanApplication(r.DB.QueryRowContext(ctx,
		"SELECT "+applicationColumns+" FROM job_applications WHERE id = $1", id))
}

func (r *Repository) Create(ctx context.Context, in ApplicationInput) (Application, error) {
	return scanApplication(r.DB.QueryRowContext(ctx,
		`INSERT INTO job_applications
		   (company, role, job_url, status, job_type, applied_at, interview_at, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 RETURNING `+applicationColumns,
		in.Company, in.Role, in.JobURL, in.Status, in.JobType,
		in.AppliedAt, in.InterviewAt, in.Notes))
}

func (r *Repository) Update(ctx context.Context, id int, in ApplicationInput) (Application, error) {
	return scanApplication(r.DB.QueryRowContext(ctx,
		`UPDATE job_applications
		 SET company=$1, role=$2, job_url=$3, status=$4, job_type=$5,
		     applied_at=$6, interview_at=$7, notes=$8
		 WHERE id=$9
		 RETURNING `+applicationColumns,
		in.Company, in.Role, in.JobURL, in.Status, in.JobType,
		in.AppliedAt, in.InterviewAt, in.Notes, id))
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	res, err := r.DB.ExecContext(ctx, `DELETE FROM job_applications WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
