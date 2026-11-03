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

type Handler struct{ svc *Service }

func NewHandler(s *Service) *Handler { return &Handler{svc: s} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /applications", h.list)
	mux.HandleFunc("POST /applications", h.create)
	mux.HandleFunc("GET /applications/{id}", h.get)
	mux.HandleFunc("PUT /applications/{id}", h.update)
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
			http.Error(w, "start_date must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}
	if v := q.Get("end_date"); v != "" {
		if f.EndDate, err = time.Parse("2006-01-02", v); err != nil {
			http.Error(w, "end_date must be YYYY-MM-DD", http.StatusBadRequest)
			return
		}
	}

	apps, err := h.svc.List(r.Context(), f)
	if err != nil {
		h.fail(w, "list", err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
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
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	a, err := h.svc.Create(r.Context(), in)
	if err != nil {
		h.fail(w, "create", err)
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var in ApplicationInput
	if err := decodeJSON(r, &in); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	a, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		h.fail(w, "update", err)
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
		http.Error(w, "not found", http.StatusNotFound)
	case errors.Is(err, ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		log.Printf("%s: %v", op, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func parseID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid id", http.StatusBadRequest)
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

func decodeJSON(r *http.Request, dst any) error {
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
