package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrontendHandlerServesAssetsAndFallsBackToIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("<div id=root></div>"), 0o644)
	os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("console.log(1)"), 0o644)
	h := frontendHandler(dir)

	cases := []struct {
		path   string
		status int
		body   string
	}{
		{"/ui/assets/app.js", 200, "console.log(1)"},
		{"/ui/applications", 200, "<div id=root>"},
		{"/ui/login", 200, "<div id=root>"},
		{"/ui/applications/7d9e/", 200, "<div id=root>"},
		{"/ui/assets/missing.js", 404, ""},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, c.path, nil))
		if rec.Code != c.status || !strings.Contains(rec.Body.String(), c.body) {
			t.Errorf("%s: got %d %q", c.path, rec.Code, rec.Body.String())
		}
	}
}

func TestFrontendHandlerDoesNotEscapeBundle(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "dist")
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0o644)
	os.WriteFile(filepath.Join(root, "secret.txt"), []byte("outside"), 0o644)
	rec := httptest.NewRecorder()
	frontendHandler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/%2e%2e/secret.txt", nil))
	if strings.Contains(rec.Body.String(), "outside") {
		t.Fatalf("served a file outside the bundle: %d %q", rec.Code, rec.Body.String())
	}
}

func TestFrontendHandlerWithoutBundle(t *testing.T) {
	rec := httptest.NewRecorder()
	frontendHandler(t.TempDir()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui/applications", nil))
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), "npm run build") {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestGoPagesStillRouteBesideUI(t *testing.T) {
	s := &Server{FrontendDir: t.TempDir(), disableAuthenticationForTests: true}
	h := s.Handler()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/ui", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/ui/applications" {
		t.Fatalf("/ui redirect: %d %v", rec.Code, rec.Header())
	}
}
