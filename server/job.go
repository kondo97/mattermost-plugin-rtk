package main

const staleHeartbeatThresholdMs = 60_000 // 60 seconds

// runJob is called by the background job scheduler every 30 seconds.
func (p *Plugin) runJob() {
	p.CleanupStaleParticipants()
}

// CleanupStaleParticipants scans all active calls and removes participants
// whose heartbeat has not been updated within the stale threshold.
func (p *Plugin) CleanupStaleParticipants() {
	// BR-23: get all active calls
	activeCalls, err := p.kvStore.GetAllActiveCalls()
	if err != nil {
		p.API.LogError("CleanupStaleParticipants: GetAllActiveCalls failed", "err", err.Error())
		return
	}

	cutoff := nowMs() - staleHeartbeatThresholdMs

	for _, session := range activeCalls {
		for _, userID := range session.Participants {
			lastBeat, err := p.kvStore.GetHeartbeat(session.ID, userID)
			if err != nil {
				p.API.LogError("CleanupStaleParticipants: GetHeartbeat failed",
					"call_id", session.ID, "user_id", userID, "err", err.Error())
				continue
			}

			// BR-24: remove if heartbeat is missing or stale
			if lastBeat == 0 || lastBeat < cutoff {
				p.API.LogInfo("CleanupStaleParticipants: removing stale participant",
					"call_id", session.ID, "user_id", userID, "last_beat", lastBeat)
				if err := p.LeaveCall(session.ID, userID); err != nil {
					p.API.LogError("CleanupStaleParticipants: LeaveCall failed",
						"call_id", session.ID, "user_id", userID, "err", err.Error())
				}
			}
		}
	}
}
