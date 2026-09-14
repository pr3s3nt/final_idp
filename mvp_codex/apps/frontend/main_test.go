package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyAPIToBackend(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/notes" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		w.WriteHeader(http.StatusTeapot)
	}))
	defer backend.Close()
	h, err := handler(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/api/notes", nil)
	response := httptest.NewRecorder()
	h.ServeHTTP(response, request)
	if response.Code != http.StatusTeapot {
		t.Fatalf("status=%d", response.Code)
	}
}
