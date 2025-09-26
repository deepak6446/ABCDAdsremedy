package service

import (
	"context"
	"country-api/internal/cache"
	"country-api/internal/domain"
	"log"
	"strings"
)

// CountryClient defines the interface for an external country data source.
// This allows us to mock the client in tests.
type CountryClient interface {
	GetCountryByName(ctx context.Context, name string) (*domain.Country, error)
}

// CountryService defines the interface for country-related business logic.
type CountryService interface {
	Search(ctx context.Context, name string) (*domain.Country, error)
}

type countryService struct {
	cache  cache.Cache
	client CountryClient // Depend on the interface, not the concrete type
}

// NewCountryService creates a new instance of the country service.
func NewCountryService(cache cache.Cache, client CountryClient) CountryService {
	return &countryService{
		cache:  cache,
		client: client,
	}
}

// ... Search function remains the same ...
// Search retrieves country information, using a cache-first strategy.
func (s *countryService) Search(ctx context.Context, name string) (*domain.Country, error) {
	// Normalize the key for caching
	cacheKey := strings.ToLower(name)

	// 1. Check cache first
	if cachedData, found := s.cache.Get(cacheKey); found {
		log.Printf("CACHE HIT: Found data for country '%s' in cache", name)
		if country, ok := cachedData.(*domain.Country); ok {
			return country, nil
		}
	}

	// 2. If not in cache, call the 3rd party API
	log.Printf("CACHE MISS: Data for country '%s' not in cache. Fetching from API.", name)
	country, err := s.client.GetCountryByName(ctx, name)
	if err != nil {
		return nil, err
	}

	// 3. Store the result in cache
	s.cache.Set(cacheKey, country)
	log.Printf("CACHE SET: Stored data for country '%s' in cache", name)

	return country, nil
}