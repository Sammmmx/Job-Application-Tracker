package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Application struct {
	ID          int       `json:"id"`
	Company     string    `json:"company"`
	Role        string    `json:"role"`
	JobURL      string    `json:"job_url"`
	Status      string    `json:"status"`
	JobType     string    `json:"job_type"`
	AppliedAt   time.Time `json:"applied_at"`
	InterviewAt time.Time `json:"interview_at"`
	Notes       string    `json:"notes"`
	CreatedAt   time.Time `json:"create_at"`
}

type ApplicationInput struct {
	Company     string    `json:"company"`
	Role        string    `json:"role"`
	JobURL      string    `json:"job_url"`
	Status      string    `json:"status"`
	JobType     string    `json:"job_type"`
	AppliedAt   time.Time `json:"applied_at"`
	InterviewAt time.Time `json:"interview_at"`
	Notes       string    `json:"notes"`
}

// JSON response helper functions

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
	}
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	// Ensure the body contains only one JSON value.
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain only one JSON value")
		}
		return err
	}

	return nil
}

func getUsersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		applications, err := getApplications(db)
		if err != nil {
			log.Printf("Failed to retrieve applications: %v", err)
			http.Error(w, "Could not retrieve applications",
				http.StatusInternalServerError)
			return
		}

		writeJSON(w, http.StatusOK, applications)
	}
}

// Database connection and initialization

func main() {
	// Load environment variables from .env file if it exists
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword("postgres", os.Getenv("PASSWORD")),
		Host:   "localhost:5432",
		Path:   "postgres",
	}
	q := u.Query()
	q.Set("sslmode", "disable")
	u.RawQuery = q.Encode()

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to PostgreSQL")

}
