package api

import (
	"context"
	"encoding/json"
	"errors"
	"country-api/internal/domain"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock for service.CountryService
type MockCountryService struct {
	SearchFunc func(ctx context.Context, name string) (*domain.Country, error)
}

func (m *MockCountryService) Search(ctx context.Context, name string) (*domain.Country, error) {
	return m.SearchFunc(ctx, name)
}

// Helper to test JSON encoding errors
type failingResponseWriter struct {
	httptest.ResponseRecorder
}

func (w *failingResponseWriter) Write(b []byte) (int, error) {
	return 0, errors.New("intentional write error")
}


func TestSearchCountry_Success(t *testing.T) {
	mockCountry := &domain.Country{
		Name:       "Japan",
		Capital:    "Tokyo",
		Currency:   "¥",
		Population: 125800000,
	}

	mockService := &MockCountryService{
		SearchFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			assert.Equal(t, "Japan", name)
			return mockCountry, nil
		},
	}

	handler := NewCountryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/countries/search?name=Japan", nil)
	rr := httptest.NewRecorder()

	handler.SearchCountry(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var respCountry domain.Country
	err := json.Unmarshal(rr.Body.Bytes(), &respCountry)
	require.NoError(t, err)
	assert.Equal(t, *mockCountry, respCountry)
}

func TestSearchCountry_NoNameQuery(t *testing.T) {
	handler := NewCountryHandler(nil) // Service won't be called
	req := httptest.NewRequest(http.MethodGet, "/api/countries/search", nil)
	rr := httptest.NewRecorder()

	handler.SearchCountry(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "Query parameter 'name' is required")
}

func TestSearchCountry_ServiceReturnsNotFound(t *testing.T) {
	mockService := &MockCountryService{
		SearchFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			return nil, errors.New("country not found")
		},
	}

	handler := NewCountryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/countries/search?name=Atlantis", nil)
	rr := httptest.NewRecorder()

	handler.SearchCountry(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "Country not found")
}

func TestSearchCountry_ServiceReturnsInternalError(t *testing.T) {
	mockService := &MockCountryService{
		SearchFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			return nil, errors.New("internal server issue")
		},
	}

	handler := NewCountryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/countries/search?name=any", nil)
	rr := httptest.NewRecorder()

	handler.SearchCountry(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Internal server error")
}

// New test to cover the JSON encoding error path
func TestSearchCountry_EncodeError(t *testing.T) {
	mockCountry := &domain.Country{Name: "Test"}

	mockService := &MockCountryService{
		SearchFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			return mockCountry, nil
		},
	}

	handler := NewCountryHandler(mockService)
	req := httptest.NewRequest(http.MethodGet, "/api/countries/search?name=Test", nil)
	
	// Use a custom response writer that will fail
	rr := &failingResponseWriter{}

	handler.SearchCountry(rr, req)

	// The header is written before the body, so the status should be OK
	// But the error path for encoding will have been triggered.
	assert.Equal(t, http.StatusOK, rr.Code)
}