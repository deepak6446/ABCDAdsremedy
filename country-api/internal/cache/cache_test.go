package cache

import (
	"sync"
	"testing"
	"fmt"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryCache_GetSet(t *testing.T) {
	c := NewInMemoryCache()
	key := "test_key"
	value := "test_value"

	// Test Set and then Get
	c.Set(key, value)
	retrieved, found := c.Get(key)

	assert.True(t, found, "Expected to find key in cache")
	assert.Equal(t, value, retrieved, "Expected value to match")
}

func TestInMemoryCache_Get_NotFound(t *testing.T) {
	c := NewInMemoryCache()
	_, found := c.Get("non_existent_key")
	assert.False(t, found, "Expected not to find key in cache")
}

func TestInMemoryCache_Overwrite(t *testing.T) {
	c := NewInMemoryCache()
	key := "key_to_overwrite"
	c.Set(key, "initial_value")
	c.Set(key, "new_value")

	retrieved, found := c.Get(key)
	assert.True(t, found)
	assert.Equal(t, "new_value", retrieved)
}

// Race Condition Test
func TestInMemoryCache_ConcurrentAccess(t *testing.T) {
	c := NewInMemoryCache()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			value := i
			c.Set(key, value)
		}(i)
	}
	wg.Wait()

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", i)
			val, found := c.Get(key)
			assert.True(t, found)
			assert.Equal(t, i, val)
		}(i)
	}
	wg.Wait()
}