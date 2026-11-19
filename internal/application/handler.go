package application

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"
)

// maxBodyBytes caps how much of a request body we are willing to read.
const maxBodyBytes = 1 << 20 // 1 MiB

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /applications", h.list)
	mux.HandleFunc("POST /applications", h.create)
	mux.HandleFunc("GET /applications/{id}", h.get)
	mux.HandleFunc("PUT /applications/{id}", h.update)
	mux.HandleFunc("PATCH /applications/{id}", h.patch)
	mux.HandleFunc("DELETE /applications/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ApplicationFilter{
		Company: q.Get("company"),
		Role:    q.Get("role"),
		Status:  q.Get("status"),
		JobType: q.Get("job_type"),
	}
	var err error
	if v := q.Get("start_date"); v != "" {
		if f.StartDate, err = time.Parse("2006-01-02", v); err != nil {
			writeError(w, http.StatusBadRequest, "start_date must be YYYY-MM-DD")
			return
		}
	}
	if v := q.Get("end_date"); v != "" {
		if f.EndDate, err = time.Parse("2006-01-02", v); err != nil {
			writeError(w, http.StatusBadRequest, "end_date must be YYYY-MM-DD")
			return
		}
	}

	var p Page
	if v := q.Get("limit"); v != "" {
		if p.Limit, err = strconv.Atoi(v); err != nil || p.Limit < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive whole number")
			return
		}
	}
	if v := q.Get("offset"); v != "" {
		if p.Offset, err = strconv.Atoi(v); err != nil || p.Offset < 0 {
			writeError(w, http.StatusBadRequest, "offset must be zero or a positive whole number")
			return
		}
	}

	result, err := h.svc.List(r.Context(), f, p)
	if err != nil {
		h.fail(w, "list", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.fail(w, "get", err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in ApplicationInput
	if err := decodeJSON(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	a, err := h.svc.Create(r.Context(), in)
	if err != nil {
		h.fail(w, "create", err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// update is PUT: replace the whole record.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ApplicationInput
	if err := decodeJSON(w, r, &in); err != nil {
		badBody(w, err)
		return
	}
	a, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		h.fail(w, "update", err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

// patch changes only the fields present in the body.
func (h *Handler) patch(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var p ApplicationPatch
	if err := decodeJSON(w, r, &p); err != nil {
		badBody(w, err)
		return
	}
	a, err := h.svc.Patch(r.Context(), id, p)
	if err != nil {
		h.fail(w, "patch", err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		h.fail(w, "delete", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// fail maps service errors to HTTP status codes in one place.
func (h *Handler) fail(w http.ResponseWriter, op string, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("%s: %v", op, err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// badBody answers a request whose body couldn't be read as the expected JSON.
func badBody(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("encode response: %v", err)
	}
}

// writeError sends every error in one shape: {"error": "message"}.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	// Stops reading after maxBodyBytes, so one huge request can't use up the server's memory.
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	if _, err := dec.Token(); err != io.EOF {
		return errors.New("request body must contain only one JSON value")
	}
	return nil
}
