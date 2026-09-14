package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/thanhnt1/final-idp/mvp-codex/internal/service"
)

type Server struct {
	token   string
	service *service.DeploymentService
	logger  *slog.Logger
}

func New(token string, deploymentService *service.DeploymentService, logger *slog.Logger) *Server {
	return &Server{token: token, service: deploymentService, logger: logger}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("POST /deployments", s.createDeployment)
	mux.HandleFunc("POST /deployments/{deploymentId}/confirm", s.confirmDeployment)
	mux.HandleFunc("GET /deployments/{deploymentId}", s.getDeployment)
	return s.authenticate(mux)
}

func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	response, err := s.service.Get(r.Context(), r.PathValue("deploymentId"))
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, errorCode(err), sanitized(err.Error()))
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) confirmDeployment(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	var request service.ConfirmDeploymentRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body is invalid")
		return
	}
	response, err := s.service.Confirm(r.Context(), r.PathValue("deploymentId"), r.Header.Get("Idempotency-Key"), request)
	if err != nil {
		code := errorCode(err)
		status := http.StatusUnprocessableEntity
		if code == "ALREADY_ACCEPTED" || code == "IDEMPOTENCY_KEY_REUSED" || code == "SCOPE_BUSY" || code == "PLAN_CHANGED" {
			status = http.StatusConflict
		}
		if errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, code, sanitized(err.Error()))
		return
	}
	if response.Code == "PLAN_CHANGED" {
		writeJSON(w, http.StatusConflict, response)
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if s.token == "" || r.Header.Get("Authorization") != "Bearer "+s.token {
			writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "valid local bearer token required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) createDeployment(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	var request service.CreateDeploymentRequest
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body is invalid")
		return
	}
	response, err := s.service.Create(r.Context(), request)
	if err != nil {
		s.logger.Warn("create deployment rejected", "error", err)
		status := http.StatusUnprocessableEntity
		if strings.Contains(err.Error(), "load application") || errors.Is(err, pgx.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeError(w, status, errorCode(err), sanitized(err.Error()))
		return
	}
	writeJSON(w, http.StatusCreated, response)
}

func errorCode(err error) string {
	message := err.Error()
	if index := strings.IndexByte(message, ':'); index > 0 {
		candidate := message[:index]
		if !strings.Contains(candidate, " ") {
			return candidate
		}
	}
	return "DEPLOYMENT_ERROR"
}

func sanitized(message string) string {
	if len(message) > 512 {
		message = message[:512]
	}
	return message
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSONStatus(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func writeJSON(w http.ResponseWriter, status int, value any) { writeJSONStatus(w, status, value) }

func writeJSONStatus(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
