// Package middleware holds HTTP wrappers that apply to every request:
// logging and panic recovery.
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

type Middleware func(http.Handler) http.Handler

// Chain wraps h so the FIRST middleware listed is the OUTERMOST one,
// i.e. the first to see the request and the last to see the response.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder remembers the status code and size of the response, because
// http.ResponseWriter doesn't let you read them back afterwards.
type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wrote {
		r.status, r.wrote = code, true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.wrote {
		r.status, r.wrote = http.StatusOK, true // Write without WriteHeader means 200
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// Unwrap lets http.ResponseController reach the real writer underneath.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// Logging writes one line per request: method, path, status, size and duration.
func Logging(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Printf("%s %s %d %dB %s", r.Method, r.URL.RequestURI(), rec.status, rec.bytes,
				time.Since(start).Round(time.Microsecond))
		})
	}
}

// Recover turns a panic inside a handler into a 500 response and a logged stack trace.
// Without it, net/http would still survive the panic, but the client would just see a
// dropped connection and the log would be far less useful.
func Recover(logger *log.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				if rec == http.ErrAbortHandler {
					panic(rec) // net/http uses this one on purpose to abort a response
				}
				logger.Printf("panic serving %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"internal error"}` + "\n"))
			}()
			next.ServeHTTP(w, r)
		})
	}
}
