package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ethpandaops/cbt-api/internal/version"
)

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health handles health check requests.
func Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Version: version.Short(),
	})
}
