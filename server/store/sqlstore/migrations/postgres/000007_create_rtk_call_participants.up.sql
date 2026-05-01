-- Prevents lost updates to participants in HA environments.
-- Normalizes rtk_call_sessions.participants (previously a JSON TEXT column) into a separate table
-- so that concurrent writes are safe via single-row INSERT/DELETE operations.
CREATE TABLE IF NOT EXISTS rtk_call_participants (
    id                   VARCHAR(26) NOT NULL,
    rtk_call_sessions_id VARCHAR(26) NOT NULL,
    user_id              VARCHAR(26) NOT NULL,
    joined_at            BIGINT      NOT NULL,
    PRIMARY KEY (id),
    UNIQUE (rtk_call_sessions_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_rtk_call_participants_call ON rtk_call_participants (rtk_call_sessions_id);
