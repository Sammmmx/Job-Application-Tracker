package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newLogger() (*log.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return log.New(&buf, "", 0), &buf
}

func TestLoggingRecordsMethodPathAndStatus(t *testing.T) {
	logger, buf := newLogger()
	h := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte("hello"))
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/applications?status=applied", nil))

	line := buf.String()
	for _, want := range []string{"POST", "/applications?status=applied", "201", "5B"} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q is missing %q", line, want)
		}
	}
	if rec.Code != http.StatusCreated || rec.Body.String() != "hello" {
		t.Errorf("response was changed by the middleware: %d %q", rec.Code, rec.Body)
	}
}

func TestLoggingDefaultsToStatus200(t *testing.T) {
	logger, buf := newLogger()
	h := Logging(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok")) // no WriteHeader call
	}))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
	if !strings.Contains(buf.String(), "200") {
		t.Errorf("log line %q should show status 200", buf.String())
	}
}

func TestRecoverTurnsPanicIntoJSON500(t *testing.T) {
	logger, buf := newLogger()
	h := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil)) // must not panic out of here

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"error"`) {
		t.Errorf("body = %q, want a JSON error", rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	if !strings.Contains(buf.String(), "boom") || !strings.Contains(buf.String(), "goroutine") {
		t.Errorf("log should hold the panic value and a stack trace, got %q", buf.String())
	}
}

func TestRecoverLetsNormalRequestsThrough(t *testing.T) {
	logger, buf := newLogger()
	h := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	if rec.Code != http.StatusNoContent || buf.Len() != 0 {
		t.Errorf("status = %d, log = %q", rec.Code, buf)
	}
}

func TestRecoverRepanicsOnErrAbortHandler(t *testing.T) {
	logger, _ := newLogger()
	h := Recover(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))
	defer func() {
		if recover() != http.ErrAbortHandler {
			t.Error("ErrAbortHandler should be passed on to net/http")
		}
	}()
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/x", nil))
}

func TestChainOrderFirstIsOutermost(t *testing.T) {
	var order []string
	tag := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+" in")
				next.ServeHTTP(w, r)
				order = append(order, name+" out")
			})
		}
	}
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), tag("A"), tag("B"))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))

	got := strings.Join(order, ", ")
	want := "A in, B in, handler, B out, A out"
	if got != want {
		t.Errorf("order = %q, want %q", got, want)
	}
}

func TestLoggingSeesThe500ProducedByRecover(t *testing.T) {
	logger, buf := newLogger()
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("x") }),
		Logging(logger), Recover(logger))
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/boom", nil))
	if !strings.Contains(buf.String(), "GET /boom 500") {
		t.Errorf("request line with 500 missing from log: %q", buf.String())
	}
}
