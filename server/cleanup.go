package main

import (
	"errors"
	"time"

	"github.com/kondo97/mattermost-plugin-rtk/server/rtkclient"
)

const cleanupInterval = 5 * time.Minute

// runCleanupLoop periodically reconciles SQL store active-call state against the
// RTK API. If a meeting no longer exists on the RTK side the call is
// force-ended in the DB and a call_ended WebSocket event is emitted so
// clients can clean up their UI.
func (p *Plugin) runCleanupLoop(stop chan struct{}) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			p.reconcileActiveCalls()
		}
	}
}

// reconcileActiveCalls iterates over every call that the SQL store considers active
// and terminates any whose RTK meeting is no longer present.
func (p *Plugin) reconcileActiveCalls() {
	if p.rtkClient == nil {
		return
	}

	callIDs, err := p.kvStore.GetActiveCallIDs()
	if err != nil {
		p.API.LogError("cleanup: GetActiveCallIDs failed", "err", err.Error())
		return
	}

	for _, callID := range callIDs {
		p.reconcileCall(callID)
	}
}

// cleanupFailThreshold is the number of consecutive cleanup cycles that must
// observe a missing RTK meeting before the call is force-ended. This prevents
// a transient RTK API 404 (e.g. during a Dyte server restart/failover) from
// terminating an otherwise-healthy call.
const cleanupFailThreshold = 2

// reconcileCall checks a single call against the RTK API and force-ends it if
// the meeting has been missing for cleanupFailThreshold consecutive cycles.
// Transient API errors are logged and skipped so that a temporary network blip
// does not accidentally terminate live calls.
func (p *Plugin) reconcileCall(callID string) {
	session, err := p.kvStore.GetCallByID(callID)
	if err != nil {
		p.API.LogWarn("cleanup: GetCallByID failed", "call_id", callID, "err", err.Error())
		return
	}
	if session == nil || session.EndAt != 0 {
		// Already ended or missing — remove stale index entry (best effort).
		_ = p.kvStore.RemoveActiveCallID(callID)
		return
	}

	_, err = p.rtkClient.GetMeetingParticipants(session.MeetingID)
	if err == nil {
		// Meeting is still alive — reset fail counter and return.
		if session.CleanupFailCount > 0 {
			p.callMu.Lock()
			defer p.callMu.Unlock()
			fresh, ferr := p.kvStore.GetCallByID(callID)
			if ferr == nil && fresh != nil && fresh.EndAt == 0 && fresh.CleanupFailCount > 0 {
				fresh.CleanupFailCount = 0
				_ = p.kvStore.SaveCall(fresh)
			}
		}
		return
	}
	if !errors.Is(err, rtkclient.ErrMeetingNotFound) {
		// Transient error — skip this cycle.
		p.API.LogWarn("cleanup: GetMeetingParticipants failed (skipping)", "call_id", callID, "meeting_id", session.MeetingID, "err", err.Error())
		return
	}

	// Meeting returned 404 — acquire lock, increment fail counter, and
	// force-end only after reaching the threshold.
	p.callMu.Lock()
	defer p.callMu.Unlock()

	// Re-fetch under the lock to avoid a TOCTOU race with EndCall/LeaveCall.
	fresh, err := p.kvStore.GetCallByID(callID)
	if err != nil || fresh == nil || fresh.EndAt != 0 {
		return
	}

	fresh.CleanupFailCount++
	if fresh.CleanupFailCount < cleanupFailThreshold {
		p.API.LogWarn("cleanup: RTK meeting not found (waiting for next cycle before force-ending)",
			"call_id", callID, "meeting_id", fresh.MeetingID, "fail_count", fresh.CleanupFailCount, "threshold", cleanupFailThreshold)
		if serr := p.kvStore.SaveCall(fresh); serr != nil {
			p.API.LogWarn("cleanup: SaveCall failed when persisting fail count (best effort)", "call_id", callID, "err", serr.Error())
		}
		return
	}

	p.API.LogInfo("cleanup: RTK meeting not found consecutively, force-ending stale call",
		"call_id", callID, "meeting_id", fresh.MeetingID, "fail_count", fresh.CleanupFailCount)
	if err := p.endCallInternal(fresh, "cleanup_loop_404"); err != nil {
		p.API.LogError("cleanup: endCallInternal failed", "call_id", callID, "err", err.Error())
	}
}
