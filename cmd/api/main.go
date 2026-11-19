package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Sammmmx/job-application-tracker/internal/application"
	"github.com/Sammmmx/job-application-tracker/internal/config"
	"github.com/Sammmmx/job-application-tracker/internal/database"
	"github.com/Sammmmx/job-application-tracker/internal/middleware"

	"github.com/joho/godotenv"
)

func main() {
	// All the work happens in run() so its deferred cleanup (closing the database)
	// really executes. log.Fatal exits immediately and would skip it.
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	_ = godotenv.Load() // optional: in production the real environment is used

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := log.New(os.Stdout, "", log.LstdFlags)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	application.NewHandler(application.NewService(application.NewRepository(db))).RegisterRoutes(mux)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if err := db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintln(w, `{"status":"database unavailable"}`)
			return
		}
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	server := &http.Server{
		Addr: ":" + cfg.Port,
		// Logging is outermost so it also records the 500 that Recover produces.
		Handler:           middleware.Chain(mux, middleware.Logging(logger), middleware.Recover(logger)),
		ErrorLog:          logger,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// ctx is cancelled when the process receives Ctrl+C or SIGTERM (what Docker sends on stop).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		logger.Printf("server listening on %s", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err // ListenAndServe only returns on its own if it failed, e.g. port in use
	case <-ctx.Done():
		logger.Println("shutting down: finishing in-flight requests...")
	}
	stop() // from here a second Ctrl+C kills the process immediately

	// Stop accepting new connections and give running requests up to 10s to finish.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Println("server stopped")
	return nil
}
