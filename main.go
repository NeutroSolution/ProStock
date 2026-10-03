package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	addr := ":" + envOr("PORT", "8080")

	// `ProStock healthcheck` probes the running server; used by the Docker
	// HEALTHCHECK since the distroless image has no curl/wget.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck("http://localhost" + addr + "/health"))
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// The pool connects lazily, so the server starts even if Postgres is not
	// up yet; /health reports the connection state.
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("invalid DATABASE_URL: %v", err)
	}
	defer db.Close()

	srv := &http.Server{
		Addr:              addr,
		Handler:           newRouter(db),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

type pinger interface {
	Ping(ctx context.Context) error
}

func newRouter(db pinger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handleHealth(db))
	return mux
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func handleHealth(db pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		resp, code := healthResponse{Status: "ok", Database: "up"}, http.StatusOK
		if err := db.Ping(ctx); err != nil {
			log.Printf("health: database ping failed: %v", err)
			resp, code = healthResponse{Status: "degraded", Database: "down"}, http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// healthcheck is a liveness probe: it succeeds whenever the server answers,
// even with 503 for a down database, so a Postgres outage doesn't make Swarm
// restart every API replica.
func healthcheck(url string) int {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusServiceUnavailable {
		return 1
	}
	return 0
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
