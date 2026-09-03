package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"pastebin/internal/store"
)

func newGetMux(s *store.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /pastes/{id}", GetHandler(s))
	return mux
}

func doGet(t *testing.T, h http.Handler, id string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/pastes/"+id, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestGetExisting(t *testing.T) {
	s := store.New()
	p, err := s.Create("hello world", "text", 0)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h := newGetMux(s)

	rec := doGet(t, h, p.ID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var got store.Paste
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if got.ID != p.ID {
		t.Fatalf("id = %q, want %q", got.ID, p.ID)
	}
	if got.Content != "hello world" {
		t.Fatalf("content = %q, want %q", got.Content, "hello world")
	}
	if got.Language != "text" {
		t.Fatalf("language = %q, want %q", got.Language, "text")
	}
	if got.ExpiresAt != nil {
		t.Fatalf("expires_at = %v, want null", got.ExpiresAt)
	}
	if got.CreatedAt.IsZero() {
		t.Fatal("created_at should be set")
	}
}

func TestGetUnknown(t *testing.T) {
	s := store.New()
	h := newGetMux(s)

	rec := doGet(t, h, "does-not-exist")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	body := strings.TrimSpace(rec.Body.String())
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("body = %q, want error object", body)
	}
}

func TestGetExpired(t *testing.T) {
	s := store.New()
	p, err := s.Create("secret content", "text", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	h := newGetMux(s)

	time.Sleep(20 * time.Millisecond)

	rec := doGet(t, h, p.ID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret content") {
		t.Fatalf("error response leaked content: %q", rec.Body.String())
	}
}

func TestGetErrorDoesNotContainContent(t *testing.T) {
	s := store.New()
	secret := "top-secret-paste-body"
	if _, err := s.Create(secret, "text", 0); err != nil {
		t.Fatalf("Create: %v", err)
	}
	h := newGetMux(s)

	rec := doGet(t, h, "unknown-id")
	body := rec.Body.String()
	if strings.Contains(body, secret) {
		t.Fatalf("error response contains content: %q", body)
	}
}
