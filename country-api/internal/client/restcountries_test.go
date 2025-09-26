package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCountryByName_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "Germany")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `[{"name":{"common":"Germany"},"capital":["Berlin"],"currencies":{"EUR":{"symbol":"€"}},"population":83240525}]`)
	}))
	defer server.Close()

	client := NewRestCountriesClient()
	client.BaseURL = server.URL // Override BaseURL for the test

	country, err := client.GetCountryByName(context.Background(), "Germany")

	require.NoError(t, err)
	require.NotNil(t, country)
	assert.Equal(t, "Germany", country.Name)
	assert.Equal(t, "Berlin", country.Capital)
	assert.Equal(t, "€", country.Currency)
	assert.Equal(t, int64(83240525), country.Population)
}
// ... all other tests in this file follow the same pattern ...
func TestGetCountryByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintln(w, `{"status": 404, "message": "Not Found"}`)
	}))
	defer server.Close()

	client := NewRestCountriesClient()
	client.BaseURL = server.URL

	_, err := client.GetCountryByName(context.Background(), "NonExistentCountry")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "country not found")
}

func TestGetCountryByName_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewRestCountriesClient()
	client.BaseURL = server.URL

	_, err := client.GetCountryByName(context.Background(), "AnyCountry")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "received non-200 status code: 500")
}

func TestGetCountryByName_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `[{"name": "Germany"`) // Malformed JSON
	}))
	defer server.Close()

	client := NewRestCountriesClient()
	client.BaseURL = server.URL

	_, err := client.GetCountryByName(context.Background(), "Germany")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestGetCountryByName_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate a slow response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRestCountriesClient()
	client.BaseURL = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.GetCountryByName(ctx, "Germany")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestMapToDomain_ErrorCases(t *testing.T) {
	testCases := []struct {
		name        string
		data        map[string]interface{}
		expectedErr string
	}{
		{"invalid name field", map[string]interface{}{"name": "not a map"}, "invalid 'name' field"},
		{"invalid common name", map[string]interface{}{"name": map[string]interface{}{"common": 123}}, "invalid 'common' name"},
		{"invalid population", map[string]interface{}{"name": map[string]interface{}{"common": "test"}, "population": "not a number"}, "invalid 'population' field"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mapToDomain(tc.data)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}