package application

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestMux() (*http.ServeMux, *fakeStore) {
	store := newFakeStore()
	mux := http.NewServeMux()
	NewHandler(NewService(store)).RegisterRoutes(mux)
	return mux, store
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

const validBody = `{"company":"Acme","role":"Backend Developer","applied_at":"2026-11-01T00:00:00Z"}`

func TestCreateHandler(t *testing.T) {
	t.Run("201 and saved record", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "POST", "/applications", validBody)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
		}
		var got Application
		if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.ID != 1 || got.Company != "Acme" || got.Status != "Applied" {
			t.Errorf("unexpected response: %+v", got)
		}
	})

	tests := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"company":`},
		{"empty body", ``},
		{"unknown field", `{"company":"A","role":"B","compnay":"typo"}`},
		{"two JSON values", validBody + validBody},
		{"missing company", `{"role":"Dev"}`},
		{"bad date format", `{"company":"A","role":"B","applied_at":"2026-11-01"}`},
	}
	for _, tc := range tests {
		t.Run("400 on "+tc.name, func(t *testing.T) {
			mux, store := newTestMux()
			rec := do(mux, "POST", "/applications", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
			if store.writes != 0 {
				t.Errorf("store was written %d times, want 0", store.writes)
			}
		})
	}
}

func TestGetHandler(t *testing.T) {
	mux, _ := newTestMux()
	do(mux, "POST", "/applications", validBody)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"existing", "/applications/1", http.StatusOK},
		{"missing", "/applications/99", http.StatusNotFound},
		{"not a number", "/applications/abc", http.StatusBadRequest},
		{"zero id", "/applications/0", http.StatusBadRequest},
		{"negative id", "/applications/-5", http.StatusBadRequest},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(mux, "GET", tc.path, "")
			if rec.Code != tc.want {
				t.Errorf("status = %d, want %d", rec.Code, tc.want)
			}
		})
	}
}

func TestListHandler(t *testing.T) {
	t.Run("empty list is [] not null", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "GET", "/applications", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
			t.Errorf("body = %s, want []", got)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
	})

	t.Run("bad start_date", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "GET", "/applications?start_date=11-01-2026", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("start after end", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "GET", "/applications?start_date=2026-12-01&end_date=2026-11-01", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestUpdateHandler(t *testing.T) {
	mux, _ := newTestMux()
	do(mux, "POST", "/applications", validBody)

	t.Run("200 and updated record", func(t *testing.T) {
		body := `{"company":"Acme","role":"Backend Developer","status":"Interview","applied_at":"2026-11-01T00:00:00Z"}`
		rec := do(mux, "PUT", "/applications/1", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}
		var got Application
		json.NewDecoder(rec.Body).Decode(&got)
		if got.Status != "Interview" {
			t.Errorf("status = %q, want Interview", got.Status)
		}
	})

	t.Run("404 for missing record", func(t *testing.T) {
		rec := do(mux, "PUT", "/applications/99", validBody)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestDeleteHandler(t *testing.T) {
	mux, _ := newTestMux()
	do(mux, "POST", "/applications", validBody)

	if rec := do(mux, "DELETE", "/applications/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("first delete: status = %d, want 204", rec.Code)
	}
	if rec := do(mux, "GET", "/applications/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("get after delete: status = %d, want 404", rec.Code)
	}
	if rec := do(mux, "DELETE", "/applications/1", ""); rec.Code != http.StatusNotFound {
		t.Errorf("second delete: status = %d, want 404", rec.Code)
	}
}
