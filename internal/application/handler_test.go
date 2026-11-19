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

// errorMessage reads {"error": "..."} out of a response and fails if the shape is wrong.
func errorMessage(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("error content-type = %q, want application/json", ct)
	}
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Error == "" {
		t.Fatalf("body %q is not {\"error\": \"...\"}", rec.Body)
	}
	return body.Error
}

func decode[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(rec.Body).Decode(&v); err != nil {
		t.Fatalf("decode %q: %v", rec.Body, err)
	}
	return v
}

const validBody = `{"company":"Acme","role":"Backend Developer","applied_at":"2026-11-01T00:00:00Z"}`

func TestCreateHandler(t *testing.T) {
	t.Run("201 and saved record", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "POST", "/applications", validBody)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body: %s", rec.Code, rec.Body)
		}
		got := decode[Application](t, rec)
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
		{"unknown status", `{"company":"A","role":"B","status":"Ghosted"}`},
		{"unknown job type", `{"company":"A","role":"B","job_type":"Hobby"}`},
		{"bad job url", `{"company":"A","role":"B","job_url":"javascript:alert(1)"}`},
	}
	for _, tc := range tests {
		t.Run("400 on "+tc.name, func(t *testing.T) {
			mux, store := newTestMux()
			rec := do(mux, "POST", "/applications", tc.body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
			errorMessage(t, rec)
			if store.writes != 0 {
				t.Errorf("store was written %d times, want 0", store.writes)
			}
		})
	}

	t.Run("status message lists the allowed values", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "POST", "/applications", `{"company":"A","role":"B","status":"Ghosted"}`)
		if msg := errorMessage(t, rec); !strings.Contains(msg, "Interview") {
			t.Errorf("message %q should list the allowed statuses", msg)
		}
	})

	t.Run("413 when the body is too large", func(t *testing.T) {
		mux, store := newTestMux()
		huge := `{"company":"Acme","role":"Dev","notes":"` + strings.Repeat("a", 2<<20) + `"}`
		rec := do(mux, "POST", "/applications", huge)
		if rec.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
		errorMessage(t, rec)
		if store.writes != 0 {
			t.Errorf("store was written %d times, want 0", store.writes)
		}
	})
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
			if tc.want >= 400 {
				errorMessage(t, rec)
			}
		})
	}
}

func TestListHandler(t *testing.T) {
	t.Run("empty list has items [] not null", func(t *testing.T) {
		mux, _ := newTestMux()
		rec := do(mux, "GET", "/applications", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"items":[]`) {
			t.Errorf("body = %s, want an empty items array", rec.Body)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %q", ct)
		}
	})

	t.Run("pagination envelope", func(t *testing.T) {
		mux, _ := newTestMux()
		for i := 0; i < 5; i++ {
			do(mux, "POST", "/applications", validBody)
		}
		rec := do(mux, "GET", "/applications?limit=2&offset=1", "")
		got := decode[ApplicationList](t, rec)
		if got.Total != 5 || got.Limit != 2 || got.Offset != 1 || len(got.Items) != 2 {
			t.Errorf("got total=%d limit=%d offset=%d items=%d", got.Total, got.Limit, got.Offset, len(got.Items))
		}
	})

	t.Run("default limit is reported", func(t *testing.T) {
		mux, _ := newTestMux()
		got := decode[ApplicationList](t, do(mux, "GET", "/applications", ""))
		if got.Limit != DefaultLimit {
			t.Errorf("limit = %d, want %d", got.Limit, DefaultLimit)
		}
	})

	bad := map[string]string{
		"bad start_date":     "start_date=11-01-2026",
		"bad end_date":       "end_date=tomorrow",
		"start after end":    "start_date=2026-12-01&end_date=2026-11-01",
		"limit not a number": "limit=abc",
		"limit zero":         "limit=0",
		"limit negative":     "limit=-3",
		"limit too large":    "limit=101",
		"offset not number":  "offset=x",
		"offset negative":    "offset=-1",
	}
	for name, query := range bad {
		t.Run("400 on "+name, func(t *testing.T) {
			mux, _ := newTestMux()
			rec := do(mux, "GET", "/applications?"+query, "")
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body)
			}
			errorMessage(t, rec)
		})
	}
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
		if got := decode[Application](t, rec); got.Status != "Interview" {
			t.Errorf("status = %q, want Interview", got.Status)
		}
	})

	t.Run("404 for missing record", func(t *testing.T) {
		rec := do(mux, "PUT", "/applications/99", validBody)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
		errorMessage(t, rec)
	})

	t.Run("400 when applied_at is left out", func(t *testing.T) {
		rec := do(mux, "PUT", "/applications/1", `{"company":"Acme","role":"Backend Developer"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body)
		}
		if msg := errorMessage(t, rec); !strings.Contains(msg, "PATCH") {
			t.Errorf("message %q should point to PATCH", msg)
		}
	})
}

func TestPatchHandler(t *testing.T) {
	setup := func() *http.ServeMux {
		mux, _ := newTestMux()
		do(mux, "POST", "/applications", `{"company":"Acme","role":"Backend Developer","job_type":"Internship",`+
			`"notes":"keep me","applied_at":"2026-11-01T00:00:00Z","interview_at":"2026-11-10T14:30:00Z"}`)
		return mux
	}

	t.Run("changes one field and keeps the rest", func(t *testing.T) {
		mux := setup()
		rec := do(mux, "PATCH", "/applications/1", `{"status":"offer"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}
		got := decode[Application](t, rec)
		if got.Status != "Offer" {
			t.Errorf("status = %q, want Offer", got.Status)
		}
		if got.Company != "Acme" || got.Notes != "keep me" || got.JobType != "Internship" ||
			got.AppliedAt.Format("2006-01-02") != "2026-11-01" || got.InterviewAt == nil {
			t.Errorf("fields that were not sent were changed: %+v", got)
		}
	})

	t.Run("null clears interview_at", func(t *testing.T) {
		mux := setup()
		rec := do(mux, "PATCH", "/applications/1", `{"interview_at":null}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body)
		}
		if got := decode[Application](t, rec); got.InterviewAt != nil {
			t.Errorf("interview_at = %v, want null", got.InterviewAt)
		}
	})

	t.Run("leaving interview_at out keeps it", func(t *testing.T) {
		mux := setup()
		got := decode[Application](t, do(mux, "PATCH", "/applications/1", `{"notes":"new"}`))
		if got.InterviewAt == nil {
			t.Error("interview_at was cleared by a patch that did not mention it")
		}
	})

	bad := []struct {
		name string
		path string
		body string
		want int
	}{
		{"empty object", "/applications/1", `{}`, 400},
		{"unknown field", "/applications/1", `{"compnay":"x"}`, 400},
		{"invalid status", "/applications/1", `{"status":"Ghosted"}`, 400},
		{"blank company", "/applications/1", `{"company":""}`, 400},
		{"bad date", "/applications/1", `{"applied_at":"yesterday"}`, 400},
		{"wrong type", "/applications/1", `{"notes":123}`, 400},
		{"bad id", "/applications/abc", `{"notes":"x"}`, 400},
		{"missing record", "/applications/99", `{"notes":"x"}`, 404},
	}
	for _, tc := range bad {
		t.Run(tc.name, func(t *testing.T) {
			mux := setup()
			rec := do(mux, "PATCH", tc.path, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d; body: %s", rec.Code, tc.want, rec.Body)
			}
			errorMessage(t, rec)
		})
	}
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
