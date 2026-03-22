package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeHTTP_UnknownRoute(t *testing.T) {
	plugin := Plugin{}
	plugin.router = plugin.initRouter()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	r.Header.Set("Mattermost-User-ID", "test-user-id")

	plugin.ServeHTTP(nil, w, r)

	assert.Equal(t, http.StatusNotFound, w.Result().StatusCode)
}

func TestServeHTTP_NoAuth(t *testing.T) {
	plugin := Plugin{}
	plugin.router = plugin.initRouter()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/config/status", nil)

	plugin.ServeHTTP(nil, w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Result().StatusCode)
}
