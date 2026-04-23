package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"

	"github.com/kondo97/mattermost-plugin-rtk/server/app"
	rtkmocks "github.com/kondo97/mattermost-plugin-rtk/server/rtkclient/mocks"
	"github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore"
	kvmocks "github.com/kondo97/mattermost-plugin-rtk/server/store/kvstore/mocks"
)

const testWebhookSecret = "test-secret"

// signBody computes the HMAC-SHA256 signature for the given body using the test secret.
func signBody(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// sendWebhook sends a POST to /api/v1/webhook/rtk with an optional HMAC signature.
func sendWebhook(t *testing.T, h *API, body []byte, signature string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/rtk", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if signature != "" {
		req.Header.Set("dyte-signature", signature)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestHandleRTKWebhook_InvalidSignature(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	body, _ := json.Marshal(rtkWebhookEvent{Event: "meeting.ended"})
	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)

	w := sendWebhook(t, h, body, "invalidsig")

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestHandleRTKWebhook_UnknownEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	body, _ := json.Marshal(rtkWebhookEvent{Event: "some.unknown.event"})
	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)

	w := sendWebhook(t, h, body, signBody(body))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_ParticipantJoined(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, mmAPI := newTestAPI(t, nil, mockStore)

	session := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"}, PostID: "post1",
	}

	event := rtkWebhookEvent{
		Event:       "meeting.participantJoined",
		Meeting:     rtkWebhookMeeting{ID: "mtg1"},
		Participant: rtkWebhookParticipant{CustomParticipantID: "user2"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(session, nil)
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)

	existingPost := &model.Post{Id: "post1", Props: model.StringInterface{"call_id": "call1"}}
	mmAPI.On("GetPost", "post1").Return(existingPost, nil)
	mmAPI.On("UpdatePost", mock.MatchedBy(func(post *model.Post) bool {
		participants, ok := post.Props["participants"].([]string)
		return ok && len(participants) == 2
	})).Return(existingPost, nil)
	mmAPI.On("PublishWebSocketEvent", app.WSEventUserJoined, mock.Anything, mock.Anything).Return()

	w := sendWebhook(t, h, body, signBody(body))

	require.Equal(t, http.StatusOK, w.Code)
	mmAPI.AssertCalled(t, "GetPost", "post1")
	mmAPI.AssertCalled(t, "UpdatePost", mock.Anything)
	mmAPI.AssertCalled(t, "PublishWebSocketEvent", app.WSEventUserJoined, mock.Anything, mock.Anything)
}

func TestHandleRTKWebhook_ParticipantJoined_SessionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	event := rtkWebhookEvent{
		Event:       "meeting.participantJoined",
		Meeting:     rtkWebhookMeeting{ID: "mtg1"},
		Participant: rtkWebhookParticipant{CustomParticipantID: "user1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(nil, nil)

	w := sendWebhook(t, h, body, signBody(body))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_ParticipantLeft(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, mmAPI := newTestAPI(t, mockRTK, mockStore)

	session := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1",
		Participants: []string{"user1", "user2"},
	}

	event := rtkWebhookEvent{
		Event:       "meeting.participantLeft",
		Meeting:     rtkWebhookMeeting{ID: "mtg1"},
		Participant: rtkWebhookParticipant{CustomParticipantID: "user1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(session, nil)
	// LeaveCall internals
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	mockStore.EXPECT().UpdateCallParticipants("call1", gomock.Any()).Return(nil)
	mmAPI.On("PublishWebSocketEvent", app.WSEventUserLeft, mock.Anything, mock.Anything).Return()

	w := sendWebhook(t, h, body, signBody(body))

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_ParticipantLeft_SessionNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	event := rtkWebhookEvent{
		Event:       "meeting.participantLeft",
		Meeting:     rtkWebhookMeeting{ID: "mtg1"},
		Participant: rtkWebhookParticipant{CustomParticipantID: "user1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(nil, nil)

	w := sendWebhook(t, h, body, signBody(body))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_MeetingEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockRTK := rtkmocks.NewMockRTKClient(ctrl)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, mmAPI := newTestAPI(t, mockRTK, mockStore)

	session := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1",
		StartAt: 1000,
	}

	event := rtkWebhookEvent{
		Event:   "meeting.ended",
		Meeting: rtkWebhookMeeting{ID: "mtg1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(session, nil)
	// Re-read inside lock (TOCTOU guard)
	mockStore.EXPECT().GetCallByID("call1").Return(session, nil)
	// endCallInternal internals
	mockStore.EXPECT().EndCall("call1", gomock.Any()).Return(nil)
	mockRTK.EXPECT().EndMeeting("mtg1").Return(nil)
	mmAPI.On("PublishWebSocketEvent", app.WSEventCallEnded, mock.Anything, mock.Anything).Return()
	mmAPI.On("GetPost", mock.Anything).Return(nil, &model.AppError{Message: "not found"}).Maybe()

	w := sendWebhook(t, h, body, signBody(body))

	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_MeetingEnded_AlreadyEndedAfterLock(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	// Initial read sees active call, but re-read inside lock sees it already ended.
	active := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", EndAt: 0}
	ended := &kvstore.CallSession{ID: "call1", ChannelID: "chan1", MeetingID: "mtg1", EndAt: 9999}

	event := rtkWebhookEvent{
		Event:   "meeting.ended",
		Meeting: rtkWebhookMeeting{ID: "mtg1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(active, nil)
	// Re-read inside lock returns already-ended session — no endCallInternal should run.
	mockStore.EXPECT().GetCallByID("call1").Return(ended, nil)

	w := sendWebhook(t, h, body, signBody(body))

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandleRTKWebhook_MeetingEnded_AlreadyEnded(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockStore := kvmocks.NewMockKVStore(ctrl)
	h, _ := newTestAPI(t, nil, mockStore)

	session := &kvstore.CallSession{
		ID: "call1", ChannelID: "chan1", MeetingID: "mtg1",
		EndAt: 9999, // already ended
	}

	event := rtkWebhookEvent{
		Event:   "meeting.ended",
		Meeting: rtkWebhookMeeting{ID: "mtg1"},
	}
	body, _ := json.Marshal(event)

	mockStore.EXPECT().GetWebhookSecret().Return(testWebhookSecret, nil)
	mockStore.EXPECT().GetCallByMeetingID("mtg1").Return(session, nil)

	w := sendWebhook(t, h, body, signBody(body))

	assert.Equal(t, http.StatusOK, w.Code)
}
