package main

import (
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"
)

//go:embed static/*
var assets embed.FS

func handler(backendURL string) (http.Handler, error) {
	target, err := url.Parse(backendURL)
	if err != nil || target.Scheme == "" || target.Host == "" {
		return nil, errors.New("BACKEND_URL must be an absolute HTTP URL")
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(r *http.Request) { original(r); r.Host = target.Host }
	static, err := fs.Sub(assets, "static")
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", proxy)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/", http.FileServer(http.FS(static)))
	return mux, nil
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	backendURL := os.Getenv("BACKEND_URL")
	h, err := handler(backendURL)
	if err != nil {
		logger.Error("configure frontend", "error", err)
		os.Exit(2)
	}
	server := &http.Server{Addr: envOr("LISTEN_ADDR", ":8080"), Handler: h, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	logger.Info("frontend listening", "address", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
