package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"idp/internal/domain"
	"idp/internal/service"
)

// EnvironmentConfigurations is the Environment Configuration Service used by
// the UC-02 JSON API. The API only loads durable state, answers catalog
// queries, stages a Secret and saves; field edits stay in the browser
// (ADR-016).
type EnvironmentConfigurations interface {
	SelectEnvironment(ctx context.Context, applicationID, environment string) (*service.ConfigurationRequirements, error)
	ResourceOutputs(ctx context.Context, applicationID, environment, workloadID, resourceID, catalogVersion, target string) (*service.ResourceOutputsView, error)
	WorkloadOutputs(ctx context.Context, applicationID, sourceWorkloadID, workloadID string) (*service.WorkloadOutputsView, error)
	StageSecret(ctx context.Context, applicationID, environment, workloadID, definitionID, value string) (string, error)
	SaveEnvironmentConfiguration(ctx context.Context, applicationID, environment string, draft service.EnvironmentConfigurationDraft) (*service.EnvironmentConfigurationDraft, error)
}

var _ EnvironmentConfigurations = (*service.ConfigurationService)(nil)

// maxSecretBytes bounds a staging request; a Secret value is small.
const maxSecretBytes = 1 << 16

func (s *Server) apiSelectEnvironment(w http.ResponseWriter, r *http.Request) {
	view, err := s.Configs.SelectEnvironment(r.Context(), r.PathValue("applicationId"), r.PathValue("environment"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) apiResourceOutputs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	view, err := s.Configs.ResourceOutputs(r.Context(), r.PathValue("applicationId"), r.PathValue("environment"),
		q.Get("workloadId"), r.PathValue("resourceId"), q.Get("catalogVersion"), q.Get("target"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func (s *Server) apiWorkloadOutputs(w http.ResponseWriter, r *http.Request) {
	view, err := s.Configs.WorkloadOutputs(r.Context(), r.PathValue("applicationId"),
		r.URL.Query().Get("workloadId"), r.PathValue("targetWorkloadId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// stageSecretRequest carries a plaintext Secret value exactly once, from the
// browser to the Secret Store. The response holds only the opaque reference.
type stageSecretRequest struct {
	WorkloadID   string `json:"workloadId"`
	DefinitionID string `json:"definitionId"`
	Value        string `json:"value"`
}

func (s *Server) apiStageSecret(w http.ResponseWriter, r *http.Request) {
	var req stageSecretRequest
	if err := decodeJSON(w, r, maxSecretBytes, &req); err != nil {
		writeError(w, err)
		return
	}
	ref, err := s.Configs.StageSecret(r.Context(), r.PathValue("applicationId"), r.PathValue("environment"),
		req.WorkloadID, req.DefinitionID, req.Value)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"secretRef": ref})
}

func (s *Server) apiSaveEnvironmentConfiguration(w http.ResponseWriter, r *http.Request) {
	var draft service.EnvironmentConfigurationDraft
	if err := decodeJSON(w, r, maxDraftBytes, &draft); err != nil {
		writeError(w, err)
		return
	}
	saved, err := s.Configs.SaveEnvironmentConfiguration(r.Context(), r.PathValue("applicationId"), r.PathValue("environment"), draft)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

// decodeJSON reads exactly one JSON object of at most limit bytes. Unknown
// fields are rejected rather than silently dropped, so a plaintext Secret sent
// in a Save request cannot pass unnoticed.
func decodeJSON(w http.ResponseWriter, r *http.Request, limit int64, target any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limit))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return domain.Reject(domain.CodeInvalidInput, "invalid request body: %v", err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return domain.Reject(domain.CodeInvalidInput, "the request body must contain one JSON object")
	}
	return nil
}
