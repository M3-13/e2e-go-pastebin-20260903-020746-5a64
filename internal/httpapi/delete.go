package httpapi

import (
	"net/http"

	"pastebin/internal/store"
)

func DeleteHandler(s *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !s.Delete(id) {
			writeError(w, http.StatusNotFound, "paste not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
