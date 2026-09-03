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

func listResponse(t *testing.T, s *store.Store) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/pastes", nil)
	rec := httptest.NewRecorder()
	ListHandler(s)(rec, req)
	return rec
}

func TestListEmptyArray(t *testing.T) {
	s := store.New()
	rec := listResponse(t, s)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body == nil {
		t.Fatal("body is null, want empty array")
	}
	if len(body) != 0 {
		t.Fatalf("len = %d, want 0", len(body))
	}
}

func TestListMultipleEntries(t *testing.T) {
	s := store.New()
	p1, err := s.Create("hello", "text", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	p2, err := s.Create("world", "go", time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec := listResponse(t, s)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(body) != 2 {
		t.Fatalf("len = %d, want 2", len(body))
	}

	ids := map[string]bool{}
	for _, item := range body {
		id, _ := item["id"].(string)
		if id == "" {
			t.Fatalf("entry missing id: %v", item)
		}
		ids[id] = true
	}
	if !ids[p1.ID] || !ids[p2.ID] {
		t.Fatalf("expected ids %q and %q in list", p1.ID, p2.ID)
	}
}

func TestListNeverContainsContent(t *testing.T) {
	s := store.New()
	_, err := s.Create("secret content", "text", 0)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	rec := listResponse(t, s)

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	for _, item := range body {
		if _, ok := item["content"]; ok {
			t.Fatalf("entry contains content field: %v", item)
		}
	}
	if raw := rec.Body.String(); strings.Contains(raw, "secret content") {
		t.Fatalf("body leaks content: %q", raw)
	}
}

func TestListExcludesExpired(t *testing.T) {
	s := store.New()
	_, err := s.Create("expired", "text", 10*time.Millisecond)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	active, err := s.Create("active", "text", time.Hour)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	time.Sleep(20 * time.Millisecond)

	rec := listResponse(t, s)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var body []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if len(body) != 1 {
		t.Fatalf("len = %d, want 1 (expired excluded)", len(body))
	}
	id, _ := body[0]["id"].(string)
	if id != active.ID {
		t.Fatalf("id = %q, want %q", id, active.ID)
	}
}
