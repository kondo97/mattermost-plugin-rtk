package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServeHTTP_UnknownRoute(t *testing.T) {
	h, _ := newTestAPI(t, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/unknown", nil)
	r.Header.Set("Mattermost-User-ID", "test-user-id")

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestServeHTTP_NoAuth(t *testing.T) {
	h, _ := newTestAPI(t, nil, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/config/status", nil)
	// No Mattermost-User-ID header

	h.ServeHTTP(w, r)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
