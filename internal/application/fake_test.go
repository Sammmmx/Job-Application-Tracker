package application

import (
	"context"
	"database/sql"
	"sort"
)

// fakeStore is an in-memory Store for tests.
type fakeStore struct {
	data   map[int]Application
	nextID int
	writes int // counts Create/Update calls
}

func newFakeStore() *fakeStore {
	return &fakeStore{data: map[int]Application{}, nextID: 1}
}

func (f *fakeStore) List(ctx context.Context, filter ApplicationFilter, p Page) ([]Application, int, error) {
	all := make([]Application, 0, len(f.data))
	for _, a := range f.data {
		all = append(all, a)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID }) // map order is random
	total := len(all)

	if p.Offset >= total {
		return []Application{}, total, nil
	}
	end := p.Offset + p.Limit
	if end > total {
		end = total
	}
	return all[p.Offset:end], total, nil
}

func (f *fakeStore) Get(ctx context.Context, id int) (Application, error) {
	a, ok := f.data[id]
	if !ok {
		return Application{}, sql.ErrNoRows
	}
	return a, nil
}

func (f *fakeStore) Create(ctx context.Context, in ApplicationInput) (Application, error) {
	f.writes++
	a := Application{
		ID:          f.nextID,
		Company:     in.Company,
		Role:        in.Role,
		JobURL:      in.JobURL,
		Status:      in.Status,
		JobType:     in.JobType,
		AppliedAt:   in.AppliedAt,
		InterviewAt: in.InterviewAt,
		Notes:       in.Notes,
	}
	f.data[a.ID] = a
	f.nextID++
	return a, nil
}

func (f *fakeStore) Update(ctx context.Context, id int, in ApplicationInput) (Application, error) {
	f.writes++
	a, ok := f.data[id]
	if !ok {
		return Application{}, sql.ErrNoRows
	}
	a.Company, a.Role, a.JobURL = in.Company, in.Role, in.JobURL
	a.Status, a.JobType, a.Notes = in.Status, in.JobType, in.Notes
	a.AppliedAt, a.InterviewAt = in.AppliedAt, in.InterviewAt
	f.data[id] = a
	return a, nil
}

func (f *fakeStore) Delete(ctx context.Context, id int) error {
	if _, ok := f.data[id]; !ok {
		return sql.ErrNoRows
	}
	delete(f.data, id)
	return nil
}
