// Package common holds startup helpers shared by the demo workloads.
package common

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Set at build time with -ldflags.
var (
	Version = "dev"
	Broken  = "false"
)

// Start exits immediately for images built as "broken", so a deployment of that
// image never becomes ready.
func Start(name string) {
	log.Printf("%s %s starting", name, Version)
	if Broken == "true" {
		log.Fatalf("%s %s is a deliberately broken build", name, Version)
	}
}

func Env(key string, required bool) string {
	v := os.Getenv(key)
	if v == "" && required {
		log.Fatalf("environment variable %s is required", key)
	}
	return v
}

// ConnectPostgres retries for about a minute, then exits so the pod never
// reports ready with a broken database configuration.
func ConnectPostgres(ctx context.Context) *pgxpool.Pool {
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=prefer",
		Env("DB_USERNAME", true), Env("DB_PASSWORD", true), Env("DB_HOST", true), Env("DB_PORT", true), Env("DB_NAME", true))
	for attempt := 1; attempt <= 30; attempt++ {
		pool, err := pgxpool.New(ctx, url)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				log.Printf("connected to postgres at %s", os.Getenv("DB_HOST"))
				return pool
			}
			pool.Close()
		}
		log.Printf("postgres not reachable (attempt %d): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	log.Fatal("giving up on postgres")
	return nil
}

func Serve(port string, mux *http.ServeMux) {
	mux.HandleFunc("GET /version", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprintln(w, Version) })
	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
