package api

import (
	"encoding/json"
	"country-api/internal/service"
	"log"
	"net/http"
	"strings"
)

// CountryHandler handles HTTP requests for country information.
type CountryHandler struct {
	service service.CountryService
}

// NewCountryHandler creates a new handler with a given service.
func NewCountryHandler(s service.CountryService) *CountryHandler {
	return &CountryHandler{
		service: s,
	}
}

// SearchCountry is the handler for the /api/countries/search endpoint.
func (h *CountryHandler) SearchCountry(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, `{"error": "Query parameter 'name' is required"}`, http.StatusBadRequest)
		return
	}

	country, err := h.service.Search(r.Context(), name)
	if err != nil {
		log.Printf("ERROR: Failed to search for country '%s': %v", name, err)
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error": "Country not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(country); err != nil {
		log.Printf("ERROR: Failed to encode response: %v", err)
	}
}