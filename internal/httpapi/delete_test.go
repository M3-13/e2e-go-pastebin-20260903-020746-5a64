package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pastebin/internal/store"
)

func newDeleteMux(s *store.Store) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("DELETE /pastes/{id}", DeleteHandler(s))
	return mux
}

func TestDeleteExistingID(t *testing.T) {
	s := store.New()
	p, err := s.Create("hello", "text", 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	mux := newDeleteMux(s)
	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+p.ID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if body := rec.Body.String(); body != "" {
		t.Fatalf("body = %q, want empty", body)
	}

	if _, ok := s.Get(p.ID); ok {
		t.Fatalf("paste still retrievable after delete")
	}
}

func TestDeleteUnknownID(t *testing.T) {
	s := store.New()

	mux := newDeleteMux(s)
	req := httptest.NewRequest(http.MethodDelete, "/pastes/nonexistent", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
}

func TestDeleteExpiredID(t *testing.T) {
	s := store.New()
	p, err := s.Create("hello", "text", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	mux := newDeleteMux(s)
	req := httptest.NewRequest(http.MethodDelete, "/pastes/"+p.ID, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	if s.Len() != 0 {
		t.Fatalf("expired paste not removed: Len = %d", s.Len())
	}
}
