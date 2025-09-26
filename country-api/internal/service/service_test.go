package service

import (
	"context"
	"errors"
	"country-api/internal/cache"
	"country-api/internal/domain"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock for cache.Cache
type MockCache struct {
	GetFunc func(key string) (interface{}, bool)
	SetFunc func(key string, value interface{})
}

func (m *MockCache) Get(key string) (interface{}, bool) { return m.GetFunc(key) }
func (m *MockCache) Set(key string, value interface{})   { m.SetFunc(key, value) }

// MockClient now implements the service.CountryClient interface
type MockClient struct {
	GetCountryByNameFunc func(ctx context.Context, name string) (*domain.Country, error)
}

func (m *MockClient) GetCountryByName(ctx context.Context, name string) (*domain.Country, error) {
	return m.GetCountryByNameFunc(ctx, name)
}

// ... the rest of the tests are now valid ...
func TestSearch_CacheHit(t *testing.T) {
	mockCountry := &domain.Country{Name: "India"}
	clientCalled := false

	mockCache := &MockCache{
		GetFunc: func(key string) (interface{}, bool) {
			assert.Equal(t, "india", key)
			return mockCountry, true
		},
		SetFunc: func(key string, value interface{}) {
			t.Fail()
		},
	}

	mockClient := &MockClient{
		GetCountryByNameFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			clientCalled = true
			return nil, nil
		},
	}

	service := NewCountryService(mockCache, mockClient)
	country, err := service.Search(context.Background(), "India")

	require.NoError(t, err)
	assert.Equal(t, mockCountry, country)
	assert.False(t, clientCalled, "Client should not be called on cache hit")
}

func TestSearch_CacheMiss_Success(t *testing.T) {
	mockCountry := &domain.Country{Name: "Germany", Population: 83000000}
	clientCalled := false
	cacheSetCalled := false

	mockCache := &MockCache{
		GetFunc: func(key string) (interface{}, bool) {
			assert.Equal(t, "germany", key)
			return nil, false // Cache miss
		},
		SetFunc: func(key string, value interface{}) {
			cacheSetCalled = true
			assert.Equal(t, "germany", key)
			assert.Equal(t, mockCountry, value)
		},
	}

	mockClient := &MockClient{
		GetCountryByNameFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			clientCalled = true
			assert.Equal(t, "Germany", name)
			return mockCountry, nil
		},
	}

	service := NewCountryService(mockCache, mockClient)
	country, err := service.Search(context.Background(), "Germany")

	require.NoError(t, err)
	assert.Equal(t, mockCountry, country)
	assert.True(t, clientCalled, "Client should be called on cache miss")
	assert.True(t, cacheSetCalled, "Cache.Set should be called on cache miss")
}

// ... other service tests remain the same and are now correct
func TestSearch_CacheMiss_ClientError(t *testing.T) {
	clientError := errors.New("API is down")
	cacheSetCalled := false

	mockCache := &MockCache{
		GetFunc: func(key string) (interface{}, bool) { return nil, false },
		SetFunc: func(key string, value interface{}) { cacheSetCalled = true },
	}

	mockClient := &MockClient{
		GetCountryByNameFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			return nil, clientError
		},
	}

	service := NewCountryService(mockCache, mockClient)
	_, err := service.Search(context.Background(), "France")

	require.Error(t, err)
	assert.Equal(t, clientError, err)
	assert.False(t, cacheSetCalled, "Cache.Set should not be called when client fails")
}

func TestSearch_ConcurrentAccess(t *testing.T) {
	mockCountry := &domain.Country{Name: "Brazil"}
	var clientCallCount int32 = 0
	var mu sync.Mutex

	mockClient := &MockClient{
		GetCountryByNameFunc: func(ctx context.Context, name string) (*domain.Country, error) {
			mu.Lock()
			clientCallCount++
			mu.Unlock()
			time.Sleep(10 * time.Millisecond)
			return mockCountry, nil
		},
	}

	realCache := cache.NewInMemoryCache()
	service := NewCountryService(realCache, mockClient)

	var wg sync.WaitGroup
	numRequests := 20
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			country, err := service.Search(context.Background(), "Brazil")
			assert.NoError(t, err)
			assert.Equal(t, "Brazil", country.Name)
		}()
	}
	wg.Wait()

	assert.True(t, clientCallCount > 0, "Client should have been called at least once")
	assert.True(t, clientCallCount <= int32(numRequests), "Client should not be called more than the number of requests")

	val, found := realCache.Get("brazil")
	assert.True(t, found)
	assert.Equal(t, mockCountry, val)
}