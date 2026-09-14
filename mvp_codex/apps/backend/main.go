package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Note struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type NoteStore interface {
	Create(context.Context, string) (Note, error)
	List(context.Context) ([]Note, error)
}

type postgresNotes struct{ pool *pgxpool.Pool }

func (s postgresNotes) Create(ctx context.Context, body string) (Note, error) {
	var note Note
	err := s.pool.QueryRow(ctx, `INSERT INTO note(body) VALUES($1) RETURNING id,body,created_at`, body).Scan(&note.ID, &note.Body, &note.CreatedAt)
	return note, err
}

func (s postgresNotes) List(ctx context.Context) ([]Note, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,body,created_at FROM note ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.Body, &note.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, note)
	}
	return result, rows.Err()
}

func handler(store NoteStore) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/notes", func(w http.ResponseWriter, r *http.Request) {
		notes, err := store.List(r.Context())
		if err != nil {
			writeError(w)
			return
		}
		writeJSON(w, http.StatusOK, notes)
	})
	mux.HandleFunc("POST /api/notes", func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Body string `json:"body"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil || input.Body == "" || len(input.Body) > 2000 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "body must contain 1-2000 characters"})
			return
		}
		note, err := store.Create(r.Context(), input.Body)
		if err != nil {
			writeError(w)
			return
		}
		writeJSON(w, http.StatusCreated, note)
	})
	return mux
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	databaseURL, err := databaseConnectionURL()
	if err != nil {
		logger.Error("database configuration is invalid", "error", err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if _, err = pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS note(id BIGSERIAL PRIMARY KEY,body TEXT NOT NULL CHECK(length(body) BETWEEN 1 AND 2000),created_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		logger.Error("migrate notes", "error", err)
		os.Exit(1)
	}
	server := &http.Server{Addr: envOr("LISTEN_ADDR", ":8080"), Handler: handler(postgresNotes{pool}), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("backend listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func databaseConnectionURL() (string, error) {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value, nil
	}
	host := os.Getenv("DB_HOST")
	port := envOr("DB_PORT", "5432")
	database := envOr("DB_NAME", "notes")
	username := envOr("DB_USER", "notes")
	password := os.Getenv("DB_PASSWORD")
	sslMode := envOr("DB_SSLMODE", "require")
	if host == "" || password == "" {
		return "", errors.New("DATABASE_URL or DB_HOST and DB_PASSWORD are required")
	}
	if sslMode != "require" && sslMode != "verify-ca" && sslMode != "verify-full" && sslMode != "disable" {
		return "", errors.New("DB_SSLMODE is not allowed")
	}
	connection := &url.URL{Scheme: "postgres", Host: host + ":" + port, Path: database, User: url.UserPassword(username, password)}
	query := connection.Query()
	query.Set("sslmode", sslMode)
	connection.RawQuery = query.Encode()
	return connection.String(), nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter) {
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "database operation failed"})
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
