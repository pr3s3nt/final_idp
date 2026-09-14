package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type memoryNotes struct{ notes []Note }

func (s *memoryNotes) Create(_ context.Context, body string) (Note, error) {
	note := Note{ID: int64(len(s.notes) + 1), Body: body, CreatedAt: time.Unix(1, 0).UTC()}
	s.notes = append(s.notes, note)
	return note, nil
}
func (s *memoryNotes) List(context.Context) ([]Note, error) { return s.notes, nil }

func TestCreateThenListNote(t *testing.T) {
	store := &memoryNotes{}
	server := httptest.NewServer(handler(store))
	defer server.Close()
	response, err := http.Post(server.URL+"/api/notes", "application/json", strings.NewReader(`{"body":"hello AWS"}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status=%d", response.StatusCode)
	}
	response, err = http.Get(server.URL + "/api/notes")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var notes []Note
	if err := json.NewDecoder(response.Body).Decode(&notes); err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Body != "hello AWS" {
		t.Fatalf("notes=%#v", notes)
	}
}

func TestDatabaseConnectionURLFromMaterializedConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "postgres.demo.svc.cluster.local")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_NAME", "notes")
	t.Setenv("DB_USER", "notes")
	t.Setenv("DB_PASSWORD", "p@ss word")
	t.Setenv("DB_SSLMODE", "disable")
	connection, err := databaseConnectionURL()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(connection, "notes:p%40ss%20word@postgres.demo.svc.cluster.local:5432/notes") || !strings.Contains(connection, "sslmode=disable") {
		t.Fatalf("connection URL was not safely encoded: %s", connection)
	}
}
