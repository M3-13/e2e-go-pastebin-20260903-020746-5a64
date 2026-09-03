package main

import (
	"encoding/json"
	"net/http"
	"os"

	"pastebin/internal/httpapi"
	"pastebin/internal/store"
)

func newHandler() http.Handler {
	s := store.New()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /pastes", httpapi.CreateHandler(s))
	mux.HandleFunc("GET /pastes/{id}", httpapi.GetHandler(s))
	mux.HandleFunc("GET /pastes", httpapi.ListHandler(s))
	mux.HandleFunc("DELETE /pastes/{id}", httpapi.DeleteHandler(s))
	mux.HandleFunc("GET /healthz", healthHandler)

	return httpapi.Recover(mux)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := http.ListenAndServe(":"+port, newHandler()); err != nil {
		panic(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
