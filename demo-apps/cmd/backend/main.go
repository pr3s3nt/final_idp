// Backend stores notes in PostgreSQL (or memory when STORAGE_MODE=memory).
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/pr3s3nt/final_idp/demo-apps/internal/common"
)

type note struct {
	ID   int64  `json:"id"`
	Text string `json:"text"`
}

func main() {
	common.Start("backend")
	ctx := context.Background()
	apiKey := os.Getenv("API_KEY")

	var pool *pgxpool.Pool
	var mu sync.Mutex
	var memory []note
	if os.Getenv("STORAGE_MODE") != "memory" {
		pool = common.ConnectPostgres(ctx)
		if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS notes (id BIGSERIAL PRIMARY KEY, text TEXT NOT NULL)`); err != nil {
			panic(err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if pool != nil {
			if err := pool.Ping(r.Context()); err != nil {
				http.Error(w, "database unavailable", http.StatusServiceUnavailable)
				return
			}
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, r *http.Request) {
		out := []note{}
		if pool == nil {
			mu.Lock()
			out = append(out, memory...)
			mu.Unlock()
		} else {
			rows, err := pool.Query(r.Context(), `SELECT id, text FROM notes ORDER BY id`)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			for rows.Next() {
				var n note
				rows.Scan(&n.ID, &n.Text)
				out = append(out, n)
			}
			rows.Close()
		}
		json.NewEncoder(w).Encode(map[string]any{"version": common.Version, "notes": out, "apiKeyConfigured": apiKey != ""})
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var n note
		if err := json.NewDecoder(r.Body).Decode(&n); err != nil || n.Text == "" {
			http.Error(w, "body must be {\"text\": \"...\"}", http.StatusBadRequest)
			return
		}
		if pool == nil {
			mu.Lock()
			n.ID = int64(len(memory) + 1)
			memory = append(memory, n)
			mu.Unlock()
		} else if err := pool.QueryRow(r.Context(), `INSERT INTO notes (text) VALUES ($1) RETURNING id`, n.Text).Scan(&n.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(n)
	})
	common.Serve("8080", mux)
}
