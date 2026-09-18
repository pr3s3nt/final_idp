package web

import (
	"context"
	"net/http"

	"idp/internal/service"
)

// ApplicationDefinitions is the Application Service used by the UC-01 JSON API.
type ApplicationDefinitions interface {
	ListApplications(ctx context.Context) ([]service.ApplicationListItem, error)
	UpdateApplication(ctx context.Context, applicationID string) (*service.ApplicationDefinitionDraft, error)
	SaveApplicationDefinition(ctx context.Context, applicationID string, draft service.ApplicationDefinitionDraft) (*service.ApplicationDefinitionDraft, error)
}

var _ ApplicationDefinitions = (*service.ApplicationService)(nil)

// maxDraftBytes bounds a Save request; a complete draft is small.
const maxDraftBytes = 1 << 20

func (s *Server) apiListApplicationDefinitions(w http.ResponseWriter, r *http.Request) {
	apps, err := s.Apps.ListApplications(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"applications": apps})
}

func (s *Server) apiGetApplicationDefinition(w http.ResponseWriter, r *http.Request) {
	draft, err := s.Apps.UpdateApplication(r.Context(), r.PathValue("applicationId"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, draft)
}

func (s *Server) apiCreateApplicationDefinition(w http.ResponseWriter, r *http.Request) {
	s.saveApplicationDefinition(w, r, "")
}

func (s *Server) apiSaveApplicationDefinitionVersion(w http.ResponseWriter, r *http.Request) {
	s.saveApplicationDefinition(w, r, r.PathValue("applicationId"))
}

func (s *Server) saveApplicationDefinition(w http.ResponseWriter, r *http.Request, applicationID string) {
	var draft service.ApplicationDefinitionDraft
	// Unknown fields (for example an image version or a Secret value) are
	// rejected rather than silently dropped.
	if err := decodeJSON(w, r, maxDraftBytes, &draft); err != nil {
		writeError(w, err)
		return
	}
	saved, err := s.Apps.SaveApplicationDefinition(r.Context(), applicationID, draft)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}
