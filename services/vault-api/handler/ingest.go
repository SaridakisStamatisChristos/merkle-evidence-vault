package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
)

type IngestHandler struct{}

func NewIngestHandler() *IngestHandler { return &IngestHandler{} }

func (h *IngestHandler) Ingest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ContentType string `json:"content_type"`
		Payload     []byte `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(400)
		return
	}
	if len(req.Payload) == 0 {
		w.WriteHeader(400)
		return
	}

	id := uuid.NewString()
	resp := map[string]interface{}{"id": id, "content_hash": "", "status": "pending"}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(202)
	json.NewEncoder(w).Encode(resp)
}
