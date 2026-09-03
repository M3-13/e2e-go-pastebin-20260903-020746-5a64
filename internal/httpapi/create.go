package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"pastebin/internal/store"
)

const maxBodyBytes = 1 << 20

type createRequest struct {
	Content          string `json:"content"`
	Language         string `json:"language"`
	ExpiresInSeconds *int   `json:"expires_in_seconds"`
}

type createResponse struct {
	ID        string     `json:"id"`
	Language  string     `json:"language"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func CreateHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
				return
			}
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		if req.Content == "" {
			writeError(w, http.StatusBadRequest, "content is required")
			return
		}

		if req.ExpiresInSeconds != nil && *req.ExpiresInSeconds <= 0 {
			writeError(w, http.StatusBadRequest, "expires_in_seconds must be greater than 0")
			return
		}

		var expiresIn time.Duration
		if req.ExpiresInSeconds != nil {
			expiresIn = time.Duration(*req.ExpiresInSeconds) * time.Second
		}

		p, err := s.Create(req.Content, req.Language, expiresIn)
		if err != nil {
			if errors.Is(err, store.ErrStoreFull) {
				writeError(w, http.StatusServiceUnavailable, "store full")
				return
			}
			writeError(w, http.StatusInternalServerError, "internal server error")
			return
		}

		writeJSON(w, http.StatusCreated, createResponse{
			ID:        p.ID,
			Language:  p.Language,
			CreatedAt: p.CreatedAt,
			ExpiresAt: p.ExpiresAt,
		})
	}
}
