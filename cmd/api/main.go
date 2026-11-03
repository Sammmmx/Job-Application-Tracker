package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Sammmmx/job-application-tracker/internal/application"
	"github.com/Sammmmx/job-application-tracker/internal/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	db, err := database.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	h := application.NewHandler(application.NewService(application.NewRepository(db)))

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server listening on :8080")
	log.Fatal(server.ListenAndServe())
}
