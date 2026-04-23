package api

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandleDismiss(t *testing.T) {
	h, mmAPI := newTestAPI(t, nil, nil)

	mmAPI.On("PublishWebSocketEvent", wsEventNotificationDismissed,
		mock.MatchedBy(func(data map[string]any) bool {
			return data["call_id"] == "call1" && data["user_id"] == "user1"
		}),
		mock.Anything,
	).Return()

	w := serveWithUser(t, h, http.MethodPost, "/api/v1/calls/call1/dismiss", "user1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
	mmAPI.AssertExpectations(t)
}
