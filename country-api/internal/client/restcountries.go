package client

import (
	"context"
	"encoding/json"
	"fmt"
	"country-api/internal/domain"
	"net/http"
	"time"
)

const (
	defaultBaseURL = "https://restcountries.com/v3.1/name"
	clientTimeout  = 10 * time.Second
)

// RestCountriesClient interacts with the REST Countries API.
type RestCountriesClient struct {
	client  *http.Client
	BaseURL string // Make BaseURL configurable
}

// NewRestCountriesClient creates a new client for the REST Countries API.
func NewRestCountriesClient() *RestCountriesClient {
	return &RestCountriesClient{
		client: &http.Client{
			Timeout: clientTimeout,
		},
		BaseURL: defaultBaseURL,
	}
}

// GetCountryByName fetches country data by name.
func (c *RestCountriesClient) GetCountryByName(ctx context.Context, name string) (*domain.Country, error) {
	url := fmt.Sprintf("%s/%s", c.BaseURL, name)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("country not found: %s", name)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	var apiResponse []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResponse) == 0 {
		return nil, fmt.Errorf("country not found: %s", name)
	}

	return mapToDomain(apiResponse[0])
}

// ... mapToDomain function remains the same ...
// mapToDomain converts the complex API response to our simple domain model.
func mapToDomain(data map[string]interface{}) (*domain.Country, error) {
	// Name
	nameMap, ok := data["name"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid 'name' field in API response")
	}
	commonName, ok := nameMap["common"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid 'common' name in API response")
	}

	// Capital
	capitalSlice, ok := data["capital"].([]interface{})
	var capitalName string
	if ok && len(capitalSlice) > 0 {
		capitalName, _ = capitalSlice[0].(string)
	}

	// Currency
	currenciesMap, ok := data["currencies"].(map[string]interface{})
	var currencySymbol string
	if ok && len(currenciesMap) > 0 {
		for _, currencyData := range currenciesMap {
			cd, ok := currencyData.(map[string]interface{})
			if ok {
				currencySymbol, _ = cd["symbol"].(string)
				break // Take the first one
			}
		}
	}

	// Population
	population, ok := data["population"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid 'population' field in API response")
	}

	return &domain.Country{
		Name:       commonName,
		Capital:    capitalName,
		Currency:   currencySymbol,
		Population: int64(population),
	}, nil
}