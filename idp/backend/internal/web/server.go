// Package web is the Web UI and the Deployment API / Deployment Query API.
package web

import (
	"embed"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"idp/internal/domain"
	"idp/internal/domain/targetadapter"
	"idp/internal/integration/imageregistry"
	"idp/internal/service"
)

//go:embed templates/*.html
var templatesFS embed.FS

type Server struct {
	Orch     *service.Orchestrator
	Query    *service.QueryService
	Registry imageregistry.Checker
	// Apps serves the UC-01 Application Definition API.
	Apps ApplicationDefinitions
	// FrontendDir is the built React bundle served under /ui/ (ADR-017).
	FrontendDir string
	tmpl        *template.Template
}

func (s *Server) Handler() http.Handler {
	s.tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"lower": strings.ToLower,
		"short": func(s string) string {
			if len(s) > 8 {
				return s[:8]
			}
			return s
		},
		"json": func(v any) string { b, _ := json.Marshal(v); return string(b) },
		"add":  func(a, b int) int { return a + b },
		"list": func(items ...string) []string { return items },
		"statusClass": func(s string) string {
			switch s {
			case "SUCCEEDED", "READY", "HEALTHY", "SYNCED":
				return "ok"
			case "FAILED", "DEGRADED":
				return "bad"
			case "RUNNING", "DEPLOYING", "CONFIRMED", "PROGRESSING", "SYNCING", "PROVISIONING":
				return "busy"
			}
			return "idle"
		},
	}).ParseFS(templatesFS, "templates/*.html"))

	mux := http.NewServeMux()
	// JSON API
	mux.HandleFunc("GET /api/applications", s.apiApplications)
	mux.HandleFunc("GET /api/applications/{app}/deployment-form", s.apiForm)
	mux.HandleFunc("GET /api/applications/{app}/deployments", s.apiHistory)
	mux.HandleFunc("POST /api/deployments", s.apiCreate)
	mux.HandleFunc("GET /api/deployments/{id}", s.apiDetail)
	mux.HandleFunc("POST /api/deployments/{id}/confirm", s.apiConfirm)
	mux.HandleFunc("POST /api/teardowns", s.apiTeardown)
	// UC-01 Application Definition API (ADR-017)
	mux.HandleFunc("GET /api/application-definitions", s.apiListApplicationDefinitions)
	mux.HandleFunc("GET /api/application-definitions/{applicationId}", s.apiGetApplicationDefinition)
	mux.HandleFunc("POST /api/application-definitions", s.apiCreateApplicationDefinition)
	mux.HandleFunc("POST /api/application-definitions/{applicationId}/versions", s.apiSaveApplicationDefinitionVersion)
	// Web UI
	mux.HandleFunc("GET /{$}", s.pageApplications)
	mux.HandleFunc("GET /apps/{app}/deploy", s.pageForm)
	mux.HandleFunc("POST /apps/{app}/deploy", s.submitForm)
	mux.HandleFunc("GET /apps/{app}/deployments", s.pageHistory)
	mux.HandleFunc("POST /apps/{app}/teardown", s.submitTeardown)
	mux.HandleFunc("GET /deployments/{id}", s.pageDeployment)
	mux.HandleFunc("POST /deployments/{id}/confirm", s.submitConfirm)
	// React web frontend (UC-01)
	mux.Handle("GET /ui/", frontendHandler(s.FrontendDir))
	mux.Handle("GET /ui", http.RedirectHandler("/ui/applications", http.StatusSeeOther))
	return logRequests(mux)
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/static") {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		h.ServeHTTP(w, r)
	})
}

// ---------------------------------------------------------------------------
// JSON API

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func writeError(w http.ResponseWriter, err error) {
	var v *domain.ValidationError
	if errors.As(err, &v) {
		status := http.StatusUnprocessableEntity
		switch {
		case domain.HasCode(err, domain.CodeNotFound):
			status = http.StatusNotFound
		case domain.HasCode(err, domain.CodeAlreadyConfirmed), domain.HasCode(err, domain.CodeDeploymentInProgress),
			domain.HasCode(err, domain.CodeDraftConflict):
			status = http.StatusConflict
		}
		writeJSON(w, status, map[string]any{"problems": v.Problems})
		return
	}
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]any{"problems": service.ProblemsOf(err)})
}

func (s *Server) apiApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.Orch.Apps.ListApplications(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apps)
}

func (s *Server) apiForm(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f, err := s.Orch.LoadDeploymentContext(r.Context(), r.PathValue("app"), q.Get("environment"), q.Get("target"), q.Get("version"), q.Get("catalogVersion"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (s *Server) apiHistory(w http.ResponseWriter, r *http.Request) {
	app, list, err := s.Query.ListDeployments(r.Context(), r.PathValue("app"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"application": app, "deployments": list})
}

func (s *Server) apiCreate(w http.ResponseWriter, r *http.Request) {
	var req service.CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, domain.Reject(domain.CodeInvalidInput, "invalid JSON: %v", err))
		return
	}
	view, err := s.Orch.CreateDeployment(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) apiDetail(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Query.GetDeploymentDetail(r.Context(), r.PathValue("id"), r.URL.Query().Get("live") != "false")
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) apiConfirm(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Overrides map[string]map[string]any `json:"overrides"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, domain.Reject(domain.CodeInvalidInput, "invalid JSON: %v", err))
			return
		}
	}
	res, err := s.Orch.ConfirmDeployment(r.Context(), r.PathValue("id"), body.Overrides)
	if err != nil {
		writeError(w, err)
		return
	}
	if res.PlanChanged {
		writeJSON(w, http.StatusConflict, map[string]any{"problems": []domain.Problem{{Code: domain.CodePlanChanged,
			Message: "the plan changed since it was created; review the rebuilt plan and confirm again"}}, "plan": res.Plan, "fingerprint": res.Fingerprint})
		return
	}
	writeJSON(w, http.StatusAccepted, res)
}

func (s *Server) apiTeardown(w http.ResponseWriter, r *http.Request) {
	var body struct{ Application, Environment, Target string }
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, domain.Reject(domain.CodeInvalidInput, "invalid JSON: %v", err))
		return
	}
	view, err := s.Orch.CreateTeardown(r.Context(), body.Application, body.Environment, body.Target)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

// ---------------------------------------------------------------------------
// Web UI

func (s *Server) render(w http.ResponseWriter, name string, data map[string]any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("render %s: %v", name, err)
	}
}

func (s *Server) pageError(w http.ResponseWriter, err error) {
	w.WriteHeader(http.StatusBadRequest)
	s.render(w, "error.html", map[string]any{"Problems": service.ProblemsOf(err)})
}

func (s *Server) pageApplications(w http.ResponseWriter, r *http.Request) {
	apps, err := s.Orch.Apps.ListApplications(r.Context())
	if err != nil {
		s.pageError(w, err)
		return
	}
	s.render(w, "applications.html", map[string]any{"Apps": apps})
}

type formRow struct {
	Workload     domain.Workload
	RunningImage string
	Tags         []string
	Selected     bool
	Tag          string
}

func (s *Server) formData(r *http.Request, app string, values url.Values, problems []domain.Problem) (map[string]any, error) {
	f, err := s.Orch.LoadDeploymentContext(r.Context(), app, values.Get("environment"), values.Get("target"), values.Get("version"), values.Get("catalogVersion"))
	if err != nil {
		return nil, err
	}
	mirror := f.RegistryMirror
	var rows []formRow
	if f.Version != nil {
		for _, wl := range f.Version.Workloads {
			row := formRow{Workload: wl, Selected: true}
			if wi, ok := f.Running[wl.ID]; ok {
				row.RunningImage = wi.ImageRepository + ":" + wi.ImageVersion
				row.Tag = wi.ImageVersion
			}
			ref := targetadapter.ResolveImage(wl.ImageRepository, "x", mirror)
			row.Tags = s.Registry.Tags(r.Context(), strings.TrimSuffix(ref, ":x"))
			sort.Sort(sort.Reverse(sort.StringSlice(row.Tags)))
			if values.Has("workloads") {
				row.Selected = contains(values["workloads"], wl.Name)
				if t := values.Get("image." + wl.Name); t != "" {
					row.Tag = t
				}
			}
			rows = append(rows, row)
		}
	}
	return map[string]any{"Form": f, "Rows": rows, "Problems": problems, "Values": values}, nil
}

func (s *Server) pageForm(w http.ResponseWriter, r *http.Request) {
	data, err := s.formData(r, r.PathValue("app"), r.URL.Query(), nil)
	if err != nil {
		s.pageError(w, err)
		return
	}
	s.render(w, "form.html", data)
}

func (s *Server) submitForm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.pageError(w, err)
		return
	}
	target := r.PostForm.Get("target")
	region := ""
	if t, reg, ok := strings.Cut(target, "|"); ok {
		target, region = t, reg
	}
	req := service.CreateRequest{Application: r.PathValue("app"), Version: r.PostForm.Get("version"), CatalogVersion: r.PostForm.Get("catalogVersion"),
		Environment: r.PostForm.Get("environment"), Target: target, Region: region,
		Workloads: r.PostForm["workloads"], Images: map[string]string{}}
	for key, v := range r.PostForm {
		if name, ok := strings.CutPrefix(key, "image."); ok && contains(req.Workloads, name) {
			req.Images[name] = v[0]
		}
	}
	view, err := s.Orch.CreateDeployment(r.Context(), req)
	if err != nil {
		values := r.PostForm
		values.Set("target", target)
		data, ferr := s.formData(r, r.PathValue("app"), values, service.ProblemsOf(err))
		if ferr != nil {
			s.pageError(w, err)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, "form.html", data)
		return
	}
	http.Redirect(w, r, "/deployments/"+view.Deployment.ID, http.StatusSeeOther)
}

func (s *Server) pageHistory(w http.ResponseWriter, r *http.Request) {
	app, list, err := s.Query.ListDeployments(r.Context(), r.PathValue("app"))
	if err != nil {
		s.pageError(w, err)
		return
	}
	s.render(w, "history.html", map[string]any{"App": app, "Deployments": list, "Problems": nil})
}

func (s *Server) submitTeardown(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.pageError(w, err)
		return
	}
	view, err := s.Orch.CreateTeardown(r.Context(), r.PathValue("app"), r.PostForm.Get("environment"), r.PostForm.Get("target"))
	if err != nil {
		app, list, _ := s.Query.ListDeployments(r.Context(), r.PathValue("app"))
		w.WriteHeader(http.StatusUnprocessableEntity)
		s.render(w, "history.html", map[string]any{"App": app, "Deployments": list, "Problems": service.ProblemsOf(err)})
		return
	}
	http.Redirect(w, r, "/deployments/"+view.Deployment.ID, http.StatusSeeOther)
}

func (s *Server) pageDeployment(w http.ResponseWriter, r *http.Request) {
	detail, err := s.Query.GetDeploymentDetail(r.Context(), r.PathValue("id"), true)
	if err != nil {
		s.pageError(w, err)
		return
	}
	s.render(w, "deployment.html", map[string]any{"D": detail, "Problems": problemsFromQuery(r), "Changed": r.URL.Query().Get("changed") != ""})
}

func problemsFromQuery(r *http.Request) []domain.Problem {
	var out []domain.Problem
	for _, p := range r.URL.Query()["problem"] {
		code, msg, _ := strings.Cut(p, "|")
		out = append(out, domain.Problem{Code: code, Message: msg})
	}
	return out
}

func (s *Server) submitConfirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.pageError(w, err)
		return
	}
	id := r.PathValue("id")
	overrides := map[string]map[string]any{}
	for key, v := range r.PostForm {
		rest, ok := strings.CutPrefix(key, "override.")
		if !ok || strings.TrimSpace(v[0]) == "" {
			continue
		}
		component, param, ok := strings.Cut(rest, ".")
		if !ok {
			continue
		}
		if overrides[component] == nil {
			overrides[component] = map[string]any{}
		}
		value := any(strings.TrimSpace(v[0]))
		if f, err := strconv.ParseFloat(v[0], 64); err == nil {
			value = f
		}
		overrides[component][param] = value
	}
	res, err := s.Orch.ConfirmDeployment(r.Context(), id, overrides)
	target := "/deployments/" + id
	switch {
	case err != nil:
		q := url.Values{}
		for _, p := range service.ProblemsOf(err) {
			q.Add("problem", p.Code+"|"+p.Message)
		}
		target += "?" + q.Encode()
	case res.PlanChanged:
		target += "?changed=1"
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
