package main

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleDismiss(t *testing.T) {
	p, api := newTestPlugin(t, nil, nil)
	p.router = p.initRouter()

	api.On("PublishWebSocketEvent", wsEventNotificationDismissed,
		mock.MatchedBy(func(data map[string]any) bool {
			return data["call_id"] == "call1" && data["user_id"] == "user1"
		}),
		mock.Anything,
	).Return()

	w := serveWithUser(t, p, http.MethodPost, "/api/v1/calls/call1/dismiss", "user1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	api.AssertExpectations(t)
}
