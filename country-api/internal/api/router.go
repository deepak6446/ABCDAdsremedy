package api

import "net/http"

// NewRouter creates and configures a new HTTP router.
func NewRouter(h *CountryHandler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/countries/search", h.SearchCountry)
	return mux
}