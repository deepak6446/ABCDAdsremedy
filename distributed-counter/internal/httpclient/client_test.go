package httpclient

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Post_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	}))
	defer server.Close()

	client := New()
	var respBody map[string]string
	err := client.Post(context.Background(), server.URL, map[string]string{"key": "value"}, &respBody)

	require.NoError(t, err)
	assert.Equal(t, "ok", respBody["status"])
}

func TestClient_Post_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := New()
	err := client.Post(context.Background(), server.URL, nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "received non-OK status code: 500")
}